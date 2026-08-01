//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"math/bits"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

// exactIntegerParamAOTAnalysis describes a guarded version of a function
// whose selected parameters retain one exact Go integer representation while
// its result remains an ordinary Clojure value. It is driven by resolved typed
// calls rather than function or workload identity.
type exactIntegerParamAOTAnalysis struct {
	paramMask uint32
	goType    reflect.Type
	ir        *compiler.TypedIR
}

func (g *Generator) prepareIntegerParameterSpecializations() {
	if !g.directLink {
		return
	}
	candidates := make(
		map[*aotSpecializationTarget]map[reflect.Type]map[uint32]struct{},
	)
	representations := []reflect.Type{
		reflect.TypeFor[int64](),
		reflect.TypeFor[int](),
	}
	for _, caller := range g.aotCallTargets {
		if caller == nil || caller.fn == nil {
			continue
		}
		ir := compiler.BuildTypedIR(caller.fn.ASTNode())
		for _, call := range ir.DirectCallSites() {
			target := g.aotCallTargets[call.Var]
			if target == nil || !target.directLinked ||
				target.arityDispatch || target.arity < 1 ||
				target.arity > 4 ||
				len(call.ArgumentTypes) != target.arity {
				continue
			}
			for _, representation := range representations {
				var mask uint32
				for index, typ := range call.ArgumentTypes {
					if typ.Kind == compiler.IRInt &&
						typ.GoType == representation {
						mask |= uint32(1) << index
					}
				}
				if mask == 0 {
					continue
				}
				if candidates[target] == nil {
					candidates[target] = make(
						map[reflect.Type]map[uint32]struct{},
					)
				}
				if candidates[target][representation] == nil {
					candidates[target][representation] =
						make(map[uint32]struct{})
				}
				candidates[target][representation][mask] = struct{}{}
			}
		}
	}

	for _, target := range g.aotCallTargets {
		if target == nil || !target.directLinked ||
			target.arityDispatch || target.arity < 1 ||
			target.arity > 4 {
			continue
		}
		if target.float64Analysis != nil ||
			target.vectorAnalysis != nil ||
			target.ownedVectorAnalysis != nil ||
			target.recordAnalysis != nil {
			continue
		}
		fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		for _, representation := range representations {
			if representation == reflect.TypeFor[int64]() &&
				target.int64Analysis != nil {
				continue
			}
			masks := candidates[target][representation]
			candidateMasks := make([]uint32, 0, len(masks))
			for mask := range masks {
				candidateMasks = append(candidateMasks, mask)
			}
			analysis := g.analyzeExactIntegerParameterRegion(
				target.fn,
				method,
				candidateMasks,
				representation,
			)
			switch representation {
			case reflect.TypeFor[int64]():
				target.int64ParamAnalysis = analysis
			case reflect.TypeFor[int]():
				target.intParamAnalysis = analysis
			}
		}
	}
}

func (g *Generator) aotSafeInt64CallSignatures() map[*lang.Var][]compiler.IRCallSignature {
	signatures := make(map[*lang.Var][]compiler.IRCallSignature)
	for vr, target := range g.aotCallTargets {
		if target == nil || !target.int64Safe {
			continue
		}
		params := make([]compiler.IRType, target.arity)
		for i := range params {
			params[i] = compiler.IRType{
				Kind:   compiler.IRInt,
				GoType: reflect.TypeOf(int64(0)),
			}
		}
		signatures[vr] = []compiler.IRCallSignature{{
			Params: params,
			Result: compiler.IRType{
				Kind:   compiler.IRInt,
				GoType: reflect.TypeOf(int64(0)),
			},
		}}
	}
	return signatures
}

func (g *Generator) analyzeExactIntegerParameterRegion(
	fn *Fn,
	method *ast.FnMethodNode,
	candidateMasks []uint32,
	representation reflect.Type,
) *exactIntegerParamAOTAnalysis {
	if !g.directLink || fn == nil || method == nil ||
		method.IsVariadic || method.FixedArity < 1 ||
		method.FixedArity > 4 ||
		(representation != reflect.TypeFor[int64]() &&
			representation != reflect.TypeFor[int]()) {
		return nil
	}
	signatures := g.aotSafeInt64CallSignatures()
	if len(candidateMasks) == 0 &&
		(representation != reflect.TypeFor[int64]() || len(signatures) == 0) {
		return nil
	}
	root := fn.ASTNode()
	base := compiler.BuildTypedIRWithOptions(
		root,
		compiler.TypedIROptions{CallSignatures: signatures},
	)
	baseScore := base.RepresentationScore()
	baseResolved := base.ResolvedCallCount()
	bestScore := baseScore
	bestBits := method.FixedArity + 1
	var best *exactIntegerParamAOTAnalysis
	limit := uint32(1) << method.FixedArity
	for mask := uint32(1); mask < limit; mask++ {
		if len(candidateMasks) != 0 {
			available := false
			for _, candidate := range candidateMasks {
				if candidate&mask == mask {
					available = true
					break
				}
			}
			if !available {
				continue
			}
		}
		parameterTypes := make(map[*lang.Symbol]compiler.IRType)
		for i, parameter := range method.Params {
			if mask&(uint32(1)<<i) == 0 {
				continue
			}
			name := parameter.Sub.(*ast.BindingNode).Name
			parameterTypes[name] = compiler.IRType{
				Kind:   compiler.IRInt,
				GoType: representation,
			}
		}
		optimized := compiler.BuildTypedIRWithOptions(
			root,
			compiler.TypedIROptions{
				CallSignatures: signatures,
				ParameterTypes: parameterTypes,
			},
		)
		score := optimized.RepresentationScore()
		if len(candidateMasks) == 0 &&
			optimized.ResolvedCallCount() <= baseResolved {
			continue
		}
		bitCount := bits.OnesCount32(mask)
		if score <= baseScore ||
			score < bestScore ||
			score == bestScore && bitCount >= bestBits {
			continue
		}
		bestScore = score
		bestBits = bitCount
		best = &exactIntegerParamAOTAnalysis{
			paramMask: mask,
			goType:    representation,
			ir:        optimized,
		}
	}
	return best
}

func exactIntegerParamAOTTypes(
	analysis *exactIntegerParamAOTAnalysis,
	arity int,
) []string {
	result := make([]string, arity)
	for index := range result {
		result[index] = "any"
		if analysis.paramMask&(uint32(1)<<index) != 0 {
			result[index] = exactIntegerGoTypeName(analysis.goType)
		}
	}
	return result
}

func exactIntegerGoTypeName(typ reflect.Type) string {
	switch typ {
	case reflect.TypeFor[int64]():
		return "int64"
	case reflect.TypeFor[int]():
		return "int"
	default:
		panic(fmt.Sprintf("unsupported exact integer type %v", typ))
	}
}

func (g *Generator) generateIntegerParameterSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn {
		return false
	}
	analyses := []*exactIntegerParamAOTAnalysis{
		target.int64ParamAnalysis,
		target.intParamAnalysis,
	}
	helpers := []string{target.int64ParamFnVar, target.intParamFnVar}
	if analyses[0] == nil && analyses[1] == nil {
		return false
	}
	for index, analysis := range analyses {
		if analysis != nil {
			g.generateExactIntegerParameterHelper(
				method,
				analysis,
				helpers[index],
			)
		}
	}

	signature := ""
	if method.FixedArity > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, method.FixedArity, signature)
	for index, analysis := range analyses {
		if analysis != nil {
			g.generateExactIntegerParameterGuard(
				analysis,
				helpers[index],
				paramNames,
			)
		}
	}
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (g *Generator) generateExactIntegerParameterHelper(
	method *ast.FnMethodNode,
	analysis *exactIntegerParamAOTAnalysis,
	helper string,
) {
	typedParams := make([]string, method.FixedArity)
	typedSignature := make([]string, method.FixedArity)
	paramTypes := make([]compiler.IRType, method.FixedArity)
	for index := range typedParams {
		typedParams[index] = g.allocateTempVar()
		typ := "any"
		if analysis.paramMask&(uint32(1)<<index) != 0 {
			typ = exactIntegerGoTypeName(analysis.goType)
			paramTypes[index] = compiler.IRType{
				Kind:   compiler.IRInt,
				GoType: analysis.goType,
			}
		}
		typedSignature[index] = typedParams[index] + " " + typ
	}
	g.writef("%s = func(%s) any {\n",
		helper,
		strings.Join(typedSignature, ", "),
	)
	plainIR := g.currentIR
	g.currentIR = analysis.ir
	g.generateFnMethodFixedWithTypes(method, typedParams, paramTypes)
	g.currentIR = plainIR
	g.writef("}\n")
}

func (g *Generator) generateExactIntegerParameterGuard(
	analysis *exactIntegerParamAOTAnalysis,
	helper string,
	paramNames []string,
) {
	typedNames := append([]string(nil), paramNames...)
	guards := make([]string, 0, bits.OnesCount32(analysis.paramMask))
	typeName := exactIntegerGoTypeName(analysis.goType)
	for i, paramName := range paramNames {
		if analysis.paramMask&(uint32(1)<<i) == 0 {
			continue
		}
		typed := g.allocateTempVar()
		ok := g.allocateTempVar()
		g.writef("%s, %s := %s.(%s)\n", typed, ok, paramName, typeName)
		typedNames[i] = typed
		guards = append(guards, ok)
	}
	if len(guards) == 0 {
		panic("empty exact integer parameter guard")
	}
	g.writef("if %s {\n", strings.Join(guards, " && "))
	g.writef("return %s(%s)\n",
		helper,
		strings.Join(typedNames, ", "),
	)
	g.writef("}\n")
}

func (g *Generator) generateAOTIntegerParameterInvoke(
	node *ast.Node,
) (string, bool) {
	if !g.directLink || g.currentIR == nil ||
		node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	target := g.aotInvokeTarget(invoke)
	if target == nil || !target.directLinked || len(invoke.Args) != target.arity {
		return "", false
	}
	analyses := []*exactIntegerParamAOTAnalysis{
		target.int64ParamAnalysis,
		target.intParamAnalysis,
	}
	helpers := []string{target.int64ParamFnVar, target.intParamFnVar}
	for analysisIndex, analysis := range analyses {
		if analysis == nil ||
			!g.exactIntegerArgumentsMatch(analysis, invoke.Args) {
			continue
		}
		args := make([]string, len(invoke.Args))
		for index, argument := range invoke.Args {
			args[index] = g.generateASTNode(argument)
		}
		result := g.allocateTempVar()
		g.writef("// direct %s parameter call %s\n",
			exactIntegerGoTypeName(analysis.goType), target.vr)
		g.writef("%s := %s(%s)\n",
			result,
			helpers[analysisIndex],
			strings.Join(args, ", "),
		)
		return result, true
	}
	return "", false
}

func (g *Generator) exactIntegerArgumentsMatch(
	analysis *exactIntegerParamAOTAnalysis,
	arguments []*ast.Node,
) bool {
	for index, argument := range arguments {
		if analysis.paramMask&(uint32(1)<<index) == 0 {
			continue
		}
		facts := g.currentIR.Facts(argument)
		if facts.Type.Kind != compiler.IRInt ||
			facts.Type.GoType != analysis.goType {
			return false
		}
		switch analysis.goType {
		case reflect.TypeFor[int64]():
			if !g.irHasInt64Representation(argument) {
				return false
			}
		case reflect.TypeFor[int]():
			if !g.irHasIntRepresentation(argument) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (g *Generator) generateAOTSafeInt64Invoke(
	node *ast.Node,
) (string, bool) {
	if !g.directLink || g.currentIR == nil ||
		node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if facts.Signature == nil || facts.Call.Var == nil ||
		facts.Signature.Result.Kind != compiler.IRInt ||
		facts.Signature.Result.GoType != reflect.TypeOf(int64(0)) {
		return "", false
	}
	target := g.aotCallTargets[facts.Call.Var]
	if target == nil || !target.int64Safe ||
		len(facts.Signature.Params) != len(
			node.Sub.(*ast.InvokeNode).Args,
		) {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	method := target.fn.ASTNode().
		Sub.(*ast.FnNode).
		Methods[0].
		Sub.(*ast.FnMethodNode)
	locals := make(
		map[*lang.Symbol]aotTypedLocal,
		len(method.Params),
	)
	for i, argument := range invoke.Args {
		if facts.Signature.Params[i].Kind != compiler.IRInt ||
			facts.Signature.Params[i].GoType != reflect.TypeOf(int64(0)) {
			return "", false
		}
		code := g.generateASTNode(argument)
		value := g.allocateTempVar()
		g.writef("%s := %s\n", value, g.irInt64Expr(argument, code))
		parameter := method.Params[i].Sub.(*ast.BindingNode).Name
		locals[parameter] = aotTypedLocal{
			name: value,
			typ:  int64AOTPrimitive,
		}
	}
	g.writef("// inline int64 call %s\n", facts.Call.Var)
	analysis := *target.int64Analysis
	analysis.uncheckedHostCalls = make(map[*ast.HostCallNode]bool)
	analysis.guardInt32Params = false
	analysis.guardInt32Loops = nil
	emitter := int64AOTEmitter{
		g:        g,
		analysis: &analysis,
	}
	return emitter.emitExpr(method.Body, locals), true
}

// int64AOTInlineCost bounds code growth for inferred primitive leaf calls.
// It deliberately measures small, general AST operations and rejects loops or
// unsupported forms rather than recognizing particular functions.
func int64AOTInlineCost(node *ast.Node, limit int) int {
	if node == nil || limit < 0 {
		return limit + 1
	}
	cost := 1
	add := func(child *ast.Node) bool {
		remaining := limit - cost
		if remaining < 0 {
			return false
		}
		cost += int64AOTInlineCost(child, remaining)
		return cost <= limit
	}

	switch node.Op {
	case ast.OpConst, ast.OpLocal:
		return cost

	case ast.OpDo:
		doNode := node.Sub.(*ast.DoNode)
		for _, statement := range doNode.Statements {
			if !add(statement) {
				return limit + 1
			}
		}
		if !add(doNode.Ret) {
			return limit + 1
		}

	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		for _, binding := range letNode.Bindings {
			if !add(binding.Sub.(*ast.BindingNode).Init) {
				return limit + 1
			}
		}
		if !add(letNode.Body) {
			return limit + 1
		}

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		if !add(ifNode.Test) || !add(ifNode.Then) || !add(ifNode.Else) {
			return limit + 1
		}

	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if call.Target != nil && !add(call.Target) {
			return limit + 1
		}
		for _, argument := range call.Args {
			if !add(argument) {
				return limit + 1
			}
		}

	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		for _, argument := range invoke.Args {
			if !add(argument) {
				return limit + 1
			}
		}

	default:
		return limit + 1
	}
	return cost
}
