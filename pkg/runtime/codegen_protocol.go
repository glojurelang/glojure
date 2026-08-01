//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

type aotProtocolPrimitiveMethod struct {
	fn        *Fn
	signature compiler.IRCallSignature
	helperVar string
}

func (g *Generator) prepareAOTProtocolTargets(vars []namedVar) {
	if !g.directLink {
		return
	}
	for _, named := range vars {
		vr := named.vr
		if !vr.IsBound() || vr.IsMacro() || vr.IsDynamic() ||
			RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) {
			continue
		}
		multiFn, ok := codegenVarValue(vr).(*lang.MultiFn)
		if !ok || !multiFn.IsProtocol() {
			continue
		}
		index := len(g.aotProtocolCallTargets)
		target := &aotProtocolCallTarget{
			vr:            vr,
			multiFnVar:    fmt.Sprintf("aotProtocolFn%d", index),
			generationVar: fmt.Sprintf("aotProtocolGeneration%d", index),
		}
		g.aotProtocolCallTargets[vr] = target
		fmt.Fprintf(
			&g.aotDeclarations,
			"var %s *lang.MultiFn\nvar %s uint64\n",
			target.multiFnVar,
			target.generationVar,
		)
		g.prepareAOTProtocolPrimitiveMethods(target, multiFn)
	}
}

func (g *Generator) prepareAOTProtocolPrimitiveMethods(
	target *aotProtocolCallTarget,
	multiFn *lang.MultiFn,
) {
	methodTable := multiFn.GetMethodTable()
	if methodTable == nil {
		return
	}
	for seq := methodTable.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(lang.IMapEntry)
		receiver, ok := protocolDispatchIRType(entry.Key())
		if !ok {
			continue
		}
		fn, ok := entry.Val().(*Fn)
		if !ok {
			continue
		}
		fnNode := fn.ASTNode().Sub.(*ast.FnNode)
		if fnNode.IsVariadic || len(fnNode.Methods) != 1 {
			continue
		}
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		if method.IsVariadic || method.FixedArity < 1 ||
			method.FixedArity > 20 {
			continue
		}
		for _, signature := range inferAOTProtocolPrimitiveSignatures(
			method,
			receiver,
		) {
			primitive := &aotProtocolPrimitiveMethod{
				fn:        fn,
				signature: signature,
				helperVar: fmt.Sprintf(
					"aotProtocolMethod%d",
					len(g.aotProtocolMethods[fn]),
				),
			}
			// Include the protocol index because method helper indices are
			// otherwise local to each generated method value.
			primitive.helperVar = fmt.Sprintf(
				"%sFor%d",
				primitive.helperVar,
				len(g.aotProtocolCallTargets)-1,
			)
			target.methods = append(target.methods, primitive)
			g.aotProtocolMethods[fn] = append(
				g.aotProtocolMethods[fn],
				primitive,
			)
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s %s\n",
				primitive.helperVar,
				protocolPrimitiveGoSignature(signature),
			)
		}
	}
}

// protocolDispatchIRType converts only exact protocol dispatch keys. Interface
// and default methods require method-selection facts rather than an exact-type
// lookup, so they conservatively retain ordinary protocol dispatch.
func protocolDispatchIRType(dispatch any) (compiler.IRType, bool) {
	if dispatch == nil {
		return compiler.IRType{Kind: compiler.IRNil, Nullable: true}, true
	}
	typ, ok := dispatch.(reflect.Type)
	if !ok || typ.Kind() == reflect.Interface {
		return compiler.IRType{}, false
	}
	result := compiler.IRType{Kind: compiler.IRDynamic, GoType: typ}
	switch typ.Kind() {
	case reflect.Bool:
		result.Kind = compiler.IRBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		result.Kind = compiler.IRInt
	case reflect.Float32, reflect.Float64:
		result.Kind = compiler.IRFloat
	case reflect.String:
		result.Kind = compiler.IRString
	}
	return result, true
}

func (g *Generator) aotProtocolCallSignatures() map[*lang.Var][]compiler.IRCallSignature {
	result := make(map[*lang.Var][]compiler.IRCallSignature)
	for vr, target := range g.aotProtocolCallTargets {
		for _, method := range target.methods {
			result[vr] = append(result[vr], method.signature)
		}
	}
	return result
}

func (g *Generator) aotProtocolPrimitiveMethodFor(
	vr *lang.Var,
	signature *compiler.IRCallSignature,
) *aotProtocolPrimitiveMethod {
	if vr == nil || signature == nil {
		return nil
	}
	target := g.aotProtocolCallTargets[vr]
	if target == nil {
		return nil
	}
	for _, method := range target.methods {
		if sameIRCallSignature(method.signature, *signature) {
			return method
		}
	}
	return nil
}

func sameIRCallSignature(
	left, right compiler.IRCallSignature,
) bool {
	if len(left.Params) != len(right.Params) || left.Result != right.Result {
		return false
	}
	for index := range left.Params {
		if left.Params[index] != right.Params[index] {
			return false
		}
	}
	return true
}

func (g *Generator) generateProtocolSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	if !g.directLink || len(g.aotProtocolCallTargets) == 0 {
		return false
	}
	plainIR := g.currentIR
	optimizedIR := compiler.BuildTypedIRWithOptions(
		fn.ASTNode(),
		compiler.TypedIROptions{
			CallSignatures: g.aotProtocolCallSignatures(),
		},
	)
	vars := optimizedIR.ResolvedCallVars()
	if len(vars) == 0 || !optimizedIR.GuardedCallsSafe() {
		return false
	}
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].String() < vars[j].String()
	})
	guards := make([]string, 0, len(vars))
	for _, vr := range vars {
		target := g.aotProtocolCallTargets[vr]
		if target == nil {
			return false
		}
		guards = append(
			guards,
			target.multiFnVar+".ProtocolGeneration() == "+
				target.generationVar,
		)
	}

	signature := ""
	if len(paramNames) > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, len(paramNames), signature)
	g.writef("if %s {\n", strings.Join(guards, " && "))
	g.currentIR = optimizedIR
	g.generateFnMethodFixed(method, paramNames)
	g.writef("}\n")
	g.currentIR = plainIR
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (g *Generator) generateAOTProtocolPrimitiveInvoke(
	node *ast.Node,
) (string, bool) {
	if g.currentIR == nil || node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if facts.Signature == nil || facts.Call.Var == nil {
		return "", false
	}
	method := g.aotProtocolPrimitiveMethodFor(
		facts.Call.Var,
		facts.Signature,
	)
	if method == nil {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	args := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		code := g.generateASTNode(argument)
		switch facts.Signature.Params[index].Kind {
		case compiler.IRInt:
			code = g.irInt64Expr(argument, code)
		case compiler.IRFloat:
			code = g.irFloat64Expr(argument, code)
		}
		args[index] = code
	}
	result := g.allocateTempVar()
	g.writef("%s := %s(%s)\n",
		result,
		method.helperVar,
		strings.Join(args, ", "),
	)
	return result, true
}

func inferAOTProtocolPrimitiveSignatures(
	method *ast.FnMethodNode,
	receiver compiler.IRType,
) []compiler.IRCallSignature {
	if method == nil || len(method.Params) != method.FixedArity ||
		method.FixedArity < 1 {
		return nil
	}
	const candidateBudget = 1 << 12
	candidates := 1
	for index := 1; index < method.FixedArity; index++ {
		candidates *= 4
		if candidates > candidateBudget {
			return nil
		}
	}
	var result []compiler.IRCallSignature
	params := make([]compiler.IRType, method.FixedArity)
	params[0] = receiver
	var visit func(int)
	visit = func(index int) {
		if index == len(params) {
			if signature, ok := analyzeAOTProtocolPrimitiveMethod(
				method,
				params,
			); ok {
				result = append(result, signature)
			}
			return
		}
		for _, kind := range []compiler.IRValueKind{
			compiler.IRDynamic,
			compiler.IRBool,
			compiler.IRInt,
			compiler.IRFloat,
		} {
			params[index] = compiler.IRType{Kind: kind}
			visit(index + 1)
		}
	}
	visit(1)
	sort.Slice(result, func(i, j int) bool {
		for index := range result[i].Params {
			if result[i].Params[index].Kind != result[j].Params[index].Kind {
				// Concrete representations precede the dynamic wildcard.
				left := result[i].Params[index].Kind
				right := result[j].Params[index].Kind
				if left == compiler.IRDynamic {
					return false
				}
				if right == compiler.IRDynamic {
					return true
				}
				return left < right
			}
		}
		return result[i].Result.Kind < result[j].Result.Kind
	})
	return result
}

func analyzeAOTProtocolPrimitiveMethod(
	method *ast.FnMethodNode,
	params []compiler.IRType,
) (compiler.IRCallSignature, bool) {
	analyzer := &primitiveAOTAnalyzer{
		allowFloat: true,
		targets:    map[*lang.Var]*aotSpecializationTarget{},
	}
	locals := make(map[*lang.Symbol]aotPrimitiveType, len(params))
	for index, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] =
			compilerIRPrimitiveType(params[index].Kind)
	}
	result := analyzer.exprType(method.Body, locals)
	if result == invalidAOTPrimitive {
		return compiler.IRCallSignature{}, false
	}
	return compiler.IRCallSignature{
		Params: append([]compiler.IRType(nil), params...),
		Result: compiler.IRType{Kind: primitiveCompilerIRType(result)},
	}, true
}

func compilerIRPrimitiveType(kind compiler.IRValueKind) aotPrimitiveType {
	switch kind {
	case compiler.IRInt:
		return int64AOTPrimitive
	case compiler.IRFloat:
		return float64AOTPrimitive
	case compiler.IRBool:
		return boolAOTPrimitive
	default:
		return invalidAOTPrimitive
	}
}

func primitiveCompilerIRType(typ aotPrimitiveType) compiler.IRValueKind {
	switch typ {
	case int64AOTPrimitive:
		return compiler.IRInt
	case float64AOTPrimitive:
		return compiler.IRFloat
	case boolAOTPrimitive:
		return compiler.IRBool
	default:
		return compiler.IRDynamic
	}
}

func protocolPrimitiveGoSignature(
	signature compiler.IRCallSignature,
) string {
	params := make([]string, len(signature.Params))
	for index, param := range signature.Params {
		params[index] = protocolPrimitiveGoType(param.Kind)
	}
	return fmt.Sprintf(
		"func(%s) %s",
		strings.Join(params, ", "),
		protocolPrimitiveGoType(signature.Result.Kind),
	)
}

func protocolPrimitiveGoType(kind compiler.IRValueKind) string {
	switch kind {
	case compiler.IRInt:
		return "int64"
	case compiler.IRFloat:
		return "float64"
	case compiler.IRBool:
		return "bool"
	default:
		return "any"
	}
}

func (g *Generator) generateAOTProtocolPrimitiveMethod(
	primitive *aotProtocolPrimitiveMethod,
) {
	fnNode := primitive.fn.ASTNode().Sub.(*ast.FnNode)
	method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
	params := make([]string, len(method.Params))
	locals := make(map[*lang.Symbol]aotTypedLocal, len(method.Params))
	for index, param := range method.Params {
		name := g.allocateTempVar()
		typ := primitive.signature.Params[index].Kind
		params[index] = name + " " + protocolPrimitiveGoType(typ)
		locals[param.Sub.(*ast.BindingNode).Name] = aotTypedLocal{
			name: name,
			typ:  compilerIRPrimitiveType(typ),
		}
	}
	g.writef("%s = func(%s) %s {\n",
		primitive.helperVar,
		strings.Join(params, ", "),
		protocolPrimitiveGoType(primitive.signature.Result.Kind),
	)
	emitter := protocolPrimitiveEmitter{
		g:         g,
		signature: primitive.signature,
	}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s\n", result)
	g.writef("}\n")
}

type protocolPrimitiveEmitter struct {
	g         *Generator
	signature compiler.IRCallSignature
}

func (e *protocolPrimitiveEmitter) analyzer() *primitiveAOTAnalyzer {
	return &primitiveAOTAnalyzer{
		allowFloat: true,
		targets:    map[*lang.Var]*aotSpecializationTarget{},
	}
}

func (e *protocolPrimitiveEmitter) emitExpr(
	node *ast.Node,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		switch value := node.Sub.(*ast.ConstNode).Value.(type) {
		case nil:
			return "nil"
		case int64:
			return "int64(" + strconv.FormatInt(value, 10) + ")"
		case float64:
			return "float64(" +
				strconv.FormatFloat(value, 'g', -1, 64) + ")"
		case bool:
			return strconv.FormatBool(value)
		}
	case ast.OpLocal:
		return locals[node.Sub.(*ast.LocalNode).Name].name
	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		nested := cloneAOTLocals(locals)
		analyzer := e.analyzer()
		types := aotLocalTypes(nested)
		for _, binding := range let.Bindings {
			binding := binding.Sub.(*ast.BindingNode)
			typ := analyzer.exprType(binding.Init, types)
			value := e.emitExpr(binding.Init, nested)
			name := e.g.allocateTempVar()
			e.g.writef("%s := %s\n", name, value)
			nested[binding.Name] = aotTypedLocal{name: name, typ: typ}
			types[binding.Name] = typ
		}
		return e.emitExpr(let.Body, nested)
	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		analyzer := e.analyzer()
		typ := analyzer.exprType(node, aotLocalTypes(locals))
		result := e.g.allocateTempVar()
		e.g.writef("var %s %s\n", result, aotGoType(typ))
		test := e.emitExpr(conditional.Test, locals)
		e.g.writef("if %s {\n", test)
		e.g.writef("%s = %s\n", result, e.emitExpr(conditional.Then, locals))
		e.g.writef("} else {\n")
		e.g.writef("%s = %s\n", result, e.emitExpr(conditional.Else, locals))
		e.g.writef("}\n")
		return result
	case ast.OpHostCall:
		return e.emitHostCall(node.Sub.(*ast.HostCallNode), locals)
	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		if invoke.Fn.Op == ast.OpVar &&
			invoke.Fn.Sub.(*ast.VarNode).Var.String() == "#'clojure.core/=" &&
			len(invoke.Args) == 2 {
			left := e.emitExpr(invoke.Args[0], locals)
			right := e.emitExpr(invoke.Args[1], locals)
			return "(" + left + " == " + right + ")"
		}
	}
	panic(fmt.Sprintf("unsupported protocol primitive expression: %v", node.Op))
}

func (e *protocolPrimitiveEmitter) emitHostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	analyzer := e.analyzer()
	localTypes := aotLocalTypes(locals)
	args := make([]string, len(call.Args))
	types := make([]aotPrimitiveType, len(call.Args))
	for index, argument := range call.Args {
		args[index] = e.emitExpr(argument, locals)
		types[index] = analyzer.exprType(argument, localTypes)
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc", "unchecked_inc":
			if types[0] == int64AOTPrimitive && name == "inc" {
				return "lang.CheckedAddInt64(" + args[0] + ", 1)"
			}
			return "(" + args[0] + " + 1)"
		case "dec", "uncheckeddec", "unchecked_dec":
			if types[0] == int64AOTPrimitive && name == "dec" {
				return "lang.CheckedSubInt64(" + args[0] + ", 1)"
			}
			return "(" + args[0] + " - 1)"
		case "minus", "unchecked_minus":
			if types[0] == int64AOTPrimitive && name == "minus" {
				return "lang.CheckedNegateInt64(" + args[0] + ")"
			}
			return "(-" + args[0] + ")"
		case "iszero":
			return "(" + args[0] + " == 0)"
		case "ispos":
			return "(" + args[0] + " > 0)"
		case "isneg":
			return "(" + args[0] + " < 0)"
		}
	}
	resultType := analyzer.hostCallType(call, localTypes)
	if resultType == float64AOTPrimitive || resultType == boolAOTPrimitive {
		if types[0] == int64AOTPrimitive &&
			(types[1] == float64AOTPrimitive || name == "divide") {
			args[0] = "float64(" + args[0] + ")"
		}
		if types[1] == int64AOTPrimitive &&
			(types[0] == float64AOTPrimitive || name == "divide") {
			args[1] = "float64(" + args[1] + ")"
		}
	}
	switch name {
	case "add", "uncheckedadd":
		if resultType == int64AOTPrimitive && name == "add" {
			return "lang.CheckedAddInt64(" + args[0] + ", " + args[1] + ")"
		}
		return "(" + args[0] + " + " + args[1] + ")"
	case "minus", "unchecked_minus":
		if resultType == int64AOTPrimitive && name == "minus" {
			return "lang.CheckedSubInt64(" + args[0] + ", " + args[1] + ")"
		}
		return "(" + args[0] + " - " + args[1] + ")"
	case "multiply", "unchecked_multiply":
		if resultType == int64AOTPrimitive && name == "multiply" {
			return "lang.CheckedMultiplyInt64(" +
				args[0] + ", " + args[1] + ")"
		}
		return "(" + args[0] + " * " + args[1] + ")"
	case "divide", "quotient":
		return "(" + args[0] + " / " + args[1] + ")"
	case "remainder":
		return "(" + args[0] + " % " + args[1] + ")"
	case "lt":
		return "(" + args[0] + " < " + args[1] + ")"
	case "lte":
		return "(" + args[0] + " <= " + args[1] + ")"
	case "gt":
		return "(" + args[0] + " > " + args[1] + ")"
	case "gte":
		return "(" + args[0] + " >= " + args[1] + ")"
	case "equiv":
		return "(" + args[0] + " == " + args[1] + ")"
	}
	panic("unsupported protocol primitive host call")
}
