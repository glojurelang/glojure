package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// int64 AOT specialization is deliberately region based. A fixed-arity
// function receives a typed fast path only when its complete result can be
// expressed with int64 arithmetic, comparisons, lets, loops, and calls to
// itself. The ordinary generated function remains as the fallback for every
// other input type.

type aotPrimitiveType uint8

const (
	invalidAOTPrimitive aotPrimitiveType = iota
	int64AOTPrimitive
	boolAOTPrimitive
)

type int64AOTAnalysis struct {
	target   *aotSpecializationTarget
	arity    int
	usesSelf bool
}

type int64AOTAnalyzer struct {
	analysis *int64AOTAnalysis
	targets  map[*lang.Var]*aotSpecializationTarget
}

func analyzeInt64AOTFunction(
	target *aotSpecializationTarget,
	method *ast.FnMethodNode,
	targets map[*lang.Var]*aotSpecializationTarget,
) *int64AOTAnalysis {
	if target == nil || method.IsVariadic || method.FixedArity > 4 {
		return nil
	}
	analysis := &int64AOTAnalysis{
		target: target,
		arity:  method.FixedArity,
	}
	analyzer := int64AOTAnalyzer{
		analysis: analysis,
		targets:  targets,
	}
	locals := make(map[*lang.Symbol]aotPrimitiveType, method.FixedArity)
	for _, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = int64AOTPrimitive
	}
	if analyzer.exprType(method.Body, locals) != int64AOTPrimitive {
		return nil
	}
	return analysis
}

func (a *int64AOTAnalyzer) exprType(
	node *ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		switch node.Sub.(*ast.ConstNode).Value.(type) {
		case int64:
			return int64AOTPrimitive
		case bool:
			return boolAOTPrimitive
		default:
			return invalidAOTPrimitive
		}

	case ast.OpLocal:
		return locals[node.Sub.(*ast.LocalNode).Name]

	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		nested := cloneAOTTypes(locals)
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			typ := a.exprType(bindingNode.Init, nested)
			if typ == invalidAOTPrimitive {
				return invalidAOTPrimitive
			}
			nested[bindingNode.Name] = typ
		}
		return a.exprType(letNode.Body, nested)

	case ast.OpLoop:
		loop := node.Sub.(*ast.LetNode)
		nested := cloneAOTTypes(locals)
		for _, binding := range loop.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			if a.exprType(bindingNode.Init, nested) != int64AOTPrimitive {
				return invalidAOTPrimitive
			}
			nested[bindingNode.Name] = int64AOTPrimitive
		}
		if a.loopTail(loop.Body, nested, loop.LoopID, len(loop.Bindings)) {
			return int64AOTPrimitive
		}
		return invalidAOTPrimitive

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		if a.exprType(ifNode.Test, locals) != boolAOTPrimitive {
			return invalidAOTPrimitive
		}
		thenType := a.exprType(ifNode.Then, locals)
		if thenType == invalidAOTPrimitive ||
			a.exprType(ifNode.Else, locals) != thenType {
			return invalidAOTPrimitive
		}
		return thenType

	case ast.OpHostCall:
		return a.hostCallType(node.Sub.(*ast.HostCallNode), locals)

	case ast.OpInvoke:
		return a.invokeType(node.Sub.(*ast.InvokeNode), locals)
	}
	return invalidAOTPrimitive
}

func (a *int64AOTAnalyzer) loopTail(
	node *ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
	loopID *lang.Symbol,
	arity int,
) bool {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpRecur:
		recur := node.Sub.(*ast.RecurNode)
		if !lang.Equals(recur.LoopID, loopID) || len(recur.Exprs) != arity {
			return false
		}
		for _, expr := range recur.Exprs {
			if a.exprType(expr, locals) != int64AOTPrimitive {
				return false
			}
		}
		return true

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		return a.exprType(ifNode.Test, locals) == boolAOTPrimitive &&
			a.loopTail(ifNode.Then, locals, loopID, arity) &&
			a.loopTail(ifNode.Else, locals, loopID, arity)

	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		nested := cloneAOTTypes(locals)
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			typ := a.exprType(bindingNode.Init, nested)
			if typ == invalidAOTPrimitive {
				return false
			}
			nested[bindingNode.Name] = typ
		}
		return a.loopTail(letNode.Body, nested, loopID, arity)
	}
	return a.exprType(node, locals) == int64AOTPrimitive
}

func (a *int64AOTAnalyzer) hostCallType(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	if !isNumbersCall(call) {
		return invalidAOTPrimitive
	}
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec",
			"minus", "unchecked_minus":
			if a.exprType(call.Args[0], locals) == int64AOTPrimitive {
				return int64AOTPrimitive
			}
		case "iszero", "ispos", "isneg":
			if a.exprType(call.Args[0], locals) == int64AOTPrimitive {
				return boolAOTPrimitive
			}
		}
		return invalidAOTPrimitive
	}
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply", "quotient", "remainder":
		if a.allInt64(call.Args, locals, 2) {
			return int64AOTPrimitive
		}
	case "lt", "lte", "gt", "gte", "equiv":
		if a.allInt64(call.Args, locals, 2) {
			return boolAOTPrimitive
		}
	}
	return invalidAOTPrimitive
}

func (a *int64AOTAnalyzer) invokeType(
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	if invoke.Fn.Op != ast.OpVar {
		return invalidAOTPrimitive
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr == a.analysis.target.vr &&
		a.allInt64(invoke.Args, locals, a.analysis.arity) {
		a.analysis.usesSelf = true
		return int64AOTPrimitive
	}
	if target := a.targets[vr]; target != nil &&
		target.int64Analysis != nil &&
		a.allInt64(invoke.Args, locals, target.arity) {
		return int64AOTPrimitive
	}
	if vr.String() == "#'clojure.core/=" && a.allInt64(invoke.Args, locals, 2) {
		return boolAOTPrimitive
	}
	return invalidAOTPrimitive
}

func (a *int64AOTAnalyzer) allInt64(
	args []*ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
	arity int,
) bool {
	if len(args) != arity {
		return false
	}
	for _, arg := range args {
		if a.exprType(arg, locals) != int64AOTPrimitive {
			return false
		}
	}
	return true
}

func cloneAOTTypes(
	locals map[*lang.Symbol]aotPrimitiveType,
) map[*lang.Symbol]aotPrimitiveType {
	copy := make(map[*lang.Symbol]aotPrimitiveType, len(locals))
	for symbol, typ := range locals {
		copy[symbol] = typ
	}
	return copy
}

func unwrapAOTDo(node *ast.Node) *ast.Node {
	if node.Op != ast.OpDo {
		return node
	}
	doNode := node.Sub.(*ast.DoNode)
	if len(doNode.Statements) != 0 {
		return node
	}
	return doNode.Ret
}

type aotTypedLocal struct {
	name string
	typ  aotPrimitiveType
}

type int64AOTEmitter struct {
	g        *Generator
	analysis *int64AOTAnalysis
	helper   string
}

func (g *Generator) generateInt64SpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn {
		return false
	}
	analysis := target.int64Analysis
	if analysis == nil {
		return false
	}

	helper := g.allocateTempVar()
	typedParams := make([]string, method.FixedArity)
	typedSignature := make([]string, method.FixedArity)
	locals := make(map[*lang.Symbol]aotTypedLocal, method.FixedArity)
	for i, param := range method.Params {
		typedParams[i] = g.allocateTempVar()
		typedSignature[i] = typedParams[i] + " int64"
		locals[param.Sub.(*ast.BindingNode).Name] = aotTypedLocal{
			name: typedParams[i],
			typ:  int64AOTPrimitive,
		}
	}

	g.writef("var %s func(%s) (int64, bool)\n",
		helper, strings.Join(typedSignature, ", "))
	g.writef("%s = func(%s) (int64, bool) {\n",
		helper, strings.Join(typedSignature, ", "))
	if analysis.usesSelf {
		varName := g.allocVarVar(
			target.vr.Namespace().Name().String(),
			target.vr.Symbol().String(),
		)
		g.writef("if %s.RootVersion() != %s {\n", varName, target.rootVersionVar)
		g.writef("return 0, false\n")
		g.writef("}\n")
	}
	emitter := int64AOTEmitter{
		g:        g,
		analysis: analysis,
		helper:   helper,
	}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s, true\n", result)
	g.writef("}\n")
	g.writef("%s = %s\n", target.int64FnVar, helper)

	arity := method.FixedArity
	signature := ""
	if arity > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n", fnVar, arity, signature)

	guardNames := make([]string, arity)
	fastArgs := make([]string, arity)
	for i, paramName := range paramNames {
		fastArgs[i] = g.allocateTempVar()
		guardNames[i] = g.allocateTempVar()
		g.writef("%s, %s := %s.(int64)\n",
			fastArgs[i], guardNames[i], paramName)
	}
	if arity > 0 {
		g.writef("if %s {\n", strings.Join(guardNames, " && "))
	}
	fastResult := g.allocateTempVar()
	fastOK := g.allocateTempVar()
	g.writef("if %s, %s := %s(%s); %s {\n",
		fastResult, fastOK, helper, strings.Join(fastArgs, ", "), fastOK)
	g.writef("return %s\n", fastResult)
	g.writef("}\n")
	if arity > 0 {
		g.writef("}\n")
	}

	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (e *int64AOTEmitter) emitExpr(
	node *ast.Node,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		switch value := node.Sub.(*ast.ConstNode).Value.(type) {
		case int64:
			return "int64(" + strconv.FormatInt(value, 10) + ")"
		case bool:
			return strconv.FormatBool(value)
		}

	case ast.OpLocal:
		return locals[node.Sub.(*ast.LocalNode).Name].name

	case ast.OpLet:
		return e.emitLet(node.Sub.(*ast.LetNode), locals)

	case ast.OpLoop:
		return e.emitLoop(node.Sub.(*ast.LetNode), locals)

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		typ := (&int64AOTAnalyzer{
			analysis: e.analysis,
			targets:  e.g.aotCallTargets,
		}).exprType(node, aotLocalTypes(locals))
		result := e.g.allocateTempVar()
		e.g.writef("var %s %s\n", result, aotGoType(typ))
		test := e.emitExpr(ifNode.Test, locals)
		e.g.writef("if %s {\n", test)
		thenExpr := e.emitExpr(ifNode.Then, locals)
		e.g.writef("%s = %s\n", result, thenExpr)
		e.g.writef("} else {\n")
		elseExpr := e.emitExpr(ifNode.Else, locals)
		e.g.writef("%s = %s\n", result, elseExpr)
		e.g.writef("}\n")
		return result

	case ast.OpHostCall:
		return e.emitHostCall(node.Sub.(*ast.HostCallNode), locals)

	case ast.OpInvoke:
		return e.emitInvoke(node.Sub.(*ast.InvokeNode), locals)
	}
	panic(fmt.Sprintf("unsupported int64 AOT expression: %v", node.Op))
}

func (e *int64AOTEmitter) emitLet(
	letNode *ast.LetNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	analyzer := int64AOTAnalyzer{
		analysis: e.analysis,
		targets:  e.g.aotCallTargets,
	}
	typ := analyzer.exprType(letNode.Body, aotLocalTypesAfterBindings(
		&analyzer, letNode.Bindings, aotLocalTypes(locals),
	))
	result := e.g.allocateTempVar()
	e.g.writef("var %s %s\n", result, aotGoType(typ))
	e.g.writef("{\n")
	nested := cloneAOTLocals(locals)
	for _, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		value := e.emitExpr(bindingNode.Init, nested)
		name := e.g.allocateTempVar()
		e.g.writef("%s := %s\n", name, value)
		nested[bindingNode.Name] = aotTypedLocal{
			name: name,
			typ:  analyzer.exprType(bindingNode.Init, aotLocalTypes(nested)),
		}
	}
	body := e.emitExpr(letNode.Body, nested)
	e.g.writef("%s = %s\n", result, body)
	e.g.writef("}\n")
	return result
}

func (e *int64AOTEmitter) emitLoop(
	loop *ast.LetNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	result := e.g.allocateTempVar()
	label := "int64_loop_" + e.g.allocateTempVar()
	e.g.writef("var %s int64\n", result)
	e.g.writef("{\n")
	nested := cloneAOTLocals(locals)
	bindings := make([]aotTypedLocal, len(loop.Bindings))
	for i, binding := range loop.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		value := e.emitExpr(bindingNode.Init, nested)
		name := e.g.allocateTempVar()
		e.g.writef("%s := %s\n", name, value)
		local := aotTypedLocal{name: name, typ: int64AOTPrimitive}
		bindings[i] = local
		nested[bindingNode.Name] = local
	}
	e.g.writef("%s:\n", label)
	e.g.writef("for {\n")
	e.emitLoopTail(loop.Body, nested, loop.LoopID, bindings, result, label)
	e.g.writef("}\n")
	e.g.writef("}\n")
	return result
}

func (e *int64AOTEmitter) emitLoopTail(
	node *ast.Node,
	locals map[*lang.Symbol]aotTypedLocal,
	loopID *lang.Symbol,
	bindings []aotTypedLocal,
	result string,
	label string,
) {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpRecur:
		recur := node.Sub.(*ast.RecurNode)
		next := make([]string, len(recur.Exprs))
		for i, expr := range recur.Exprs {
			value := e.emitExpr(expr, locals)
			next[i] = e.g.allocateTempVar()
			e.g.writef("%s := %s\n", next[i], value)
		}
		for i, binding := range bindings {
			e.g.writef("%s = %s\n", binding.name, next[i])
		}
		e.g.writef("continue %s\n", label)

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		test := e.emitExpr(ifNode.Test, locals)
		e.g.writef("if %s {\n", test)
		e.emitLoopTail(ifNode.Then, locals, loopID, bindings, result, label)
		e.g.writef("} else {\n")
		e.emitLoopTail(ifNode.Else, locals, loopID, bindings, result, label)
		e.g.writef("}\n")

	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		e.g.writef("{\n")
		nested := cloneAOTLocals(locals)
		analyzer := int64AOTAnalyzer{
			analysis: e.analysis,
			targets:  e.g.aotCallTargets,
		}
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			value := e.emitExpr(bindingNode.Init, nested)
			name := e.g.allocateTempVar()
			e.g.writef("%s := %s\n", name, value)
			nested[bindingNode.Name] = aotTypedLocal{
				name: name,
				typ:  analyzer.exprType(bindingNode.Init, aotLocalTypes(nested)),
			}
		}
		e.emitLoopTail(letNode.Body, nested, loopID, bindings, result, label)
		e.g.writef("}\n")

	default:
		value := e.emitExpr(node, locals)
		e.g.writef("%s = %s\n", result, value)
		e.g.writef("break %s\n", label)
	}
}

func (e *int64AOTEmitter) emitHostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = e.emitExpr(arg, locals)
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc":
			return "lang.CheckedAddInt64(" + args[0] + ", 1)"
		case "unchecked_inc":
			return "(" + args[0] + " + 1)"
		case "dec":
			return "lang.CheckedSubInt64(" + args[0] + ", 1)"
		case "uncheckeddec", "unchecked_dec":
			return "(" + args[0] + " - 1)"
		case "minus":
			return "lang.CheckedNegateInt64(" + args[0] + ")"
		case "unchecked_minus":
			return "(-" + args[0] + ")"
		case "iszero":
			return "(" + args[0] + " == 0)"
		case "ispos":
			return "(" + args[0] + " > 0)"
		case "isneg":
			return "(" + args[0] + " < 0)"
		}
	}
	switch name {
	case "add":
		return "lang.CheckedAddInt64(" + args[0] + ", " + args[1] + ")"
	case "uncheckedadd":
		return "(" + args[0] + " + " + args[1] + ")"
	case "minus":
		return "lang.CheckedSubInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_minus":
		return "(" + args[0] + " - " + args[1] + ")"
	case "multiply":
		return "lang.CheckedMultiplyInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_multiply":
		return "(" + args[0] + " * " + args[1] + ")"
	case "quotient":
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
	panic("unsupported int64 AOT host call")
}

func (e *int64AOTEmitter) emitInvoke(
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	args := make([]string, len(invoke.Args))
	for i, arg := range invoke.Args {
		args[i] = e.emitExpr(arg, locals)
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.String() == "#'clojure.core/=" {
		return "(" + args[0] + " == " + args[1] + ")"
	}
	helper := e.helper
	if vr != e.analysis.target.vr {
		target := e.g.aotCallTargets[vr]
		varName := e.g.allocVarVar(
			vr.Namespace().Name().String(),
			vr.Symbol().String(),
		)
		e.g.writef("if %s.RootVersion() != %s {\n",
			varName, target.rootVersionVar)
		e.g.writef("return 0, false\n")
		e.g.writef("}\n")
		helper = target.int64FnVar
	}
	result := e.g.allocateTempVar()
	ok := e.g.allocateTempVar()
	e.g.writef("%s, %s := %s(%s)\n",
		result, ok, helper, strings.Join(args, ", "))
	e.g.writef("if !%s {\n", ok)
	e.g.writef("return 0, false\n")
	e.g.writef("}\n")
	return result
}

func cloneAOTLocals(
	locals map[*lang.Symbol]aotTypedLocal,
) map[*lang.Symbol]aotTypedLocal {
	copy := make(map[*lang.Symbol]aotTypedLocal, len(locals))
	for symbol, local := range locals {
		copy[symbol] = local
	}
	return copy
}

func aotLocalTypes(
	locals map[*lang.Symbol]aotTypedLocal,
) map[*lang.Symbol]aotPrimitiveType {
	types := make(map[*lang.Symbol]aotPrimitiveType, len(locals))
	for symbol, local := range locals {
		types[symbol] = local.typ
	}
	return types
}

func aotLocalTypesAfterBindings(
	analyzer *int64AOTAnalyzer,
	bindings []*ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
) map[*lang.Symbol]aotPrimitiveType {
	for _, binding := range bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		locals[bindingNode.Name] = analyzer.exprType(bindingNode.Init, locals)
	}
	return locals
}

func aotGoType(typ aotPrimitiveType) string {
	if typ == boolAOTPrimitive {
		return "bool"
	}
	return "int64"
}
