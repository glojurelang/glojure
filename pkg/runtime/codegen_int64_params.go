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

// int64ParamAOTAnalysis describes a guarded version of a function whose
// selected parameters have concrete int64 representations while its result
// remains an ordinary Clojure value. It is intentionally driven by resolved
// typed calls rather than by function or workload identity.
type int64ParamAOTAnalysis struct {
	paramMask uint32
	ir        *compiler.TypedIR
}

func (g *Generator) prepareInt64ParameterSpecializations() {
	if !g.directLink {
		return
	}
	candidates := make(
		map[*aotSpecializationTarget]map[uint32]struct{},
	)
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
			var mask uint32
			for index, typ := range call.ArgumentTypes {
				if typ.Kind == compiler.IRInt &&
					typ.GoType == reflect.TypeOf(int64(0)) {
					mask |= uint32(1) << index
				}
			}
			if mask == 0 {
				continue
			}
			if candidates[target] == nil {
				candidates[target] = make(map[uint32]struct{})
			}
			candidates[target][mask] = struct{}{}
		}
	}

	for _, target := range g.aotCallTargets {
		if target == nil || !target.directLinked ||
			target.arityDispatch || target.arity < 1 ||
			target.arity > 4 {
			continue
		}
		if target.int64Analysis != nil ||
			target.float64Analysis != nil ||
			target.vectorAnalysis != nil ||
			target.ownedVectorAnalysis != nil ||
			target.recordAnalysis != nil {
			continue
		}
		masks := candidates[target]
		candidateMasks := make([]uint32, 0, len(masks))
		for mask := range masks {
			candidateMasks = append(candidateMasks, mask)
		}
		fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		target.int64ParamAnalysis = g.analyzeInt64ParameterRegion(
			target.fn,
			method,
			candidateMasks,
		)
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

func (g *Generator) analyzeInt64ParameterRegion(
	fn *Fn,
	method *ast.FnMethodNode,
	candidateMasks []uint32,
) *int64ParamAOTAnalysis {
	if !g.directLink || fn == nil || method == nil ||
		method.IsVariadic || method.FixedArity < 1 ||
		method.FixedArity > 4 {
		return nil
	}
	signatures := g.aotSafeInt64CallSignatures()
	if len(candidateMasks) == 0 && len(signatures) == 0 {
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
	var best *int64ParamAOTAnalysis
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
				GoType: reflect.TypeOf(int64(0)),
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
		best = &int64ParamAOTAnalysis{
			paramMask: mask,
			ir:        optimized,
		}
	}
	return best
}

func int64ParamAOTTypes(mask uint32, arity int) []string {
	result := make([]string, arity)
	for index := range result {
		result[index] = "any"
		if mask&(uint32(1)<<index) != 0 {
			result[index] = "int64"
		}
	}
	return result
}

func (g *Generator) generateInt64ParameterSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn {
		return false
	}
	analysis := target.int64ParamAnalysis
	if analysis == nil {
		return false
	}

	typedParams := make([]string, method.FixedArity)
	typedSignature := make([]string, method.FixedArity)
	paramTypes := make([]compiler.IRValueKind, method.FixedArity)
	for index := range typedParams {
		typedParams[index] = g.allocateTempVar()
		typ := "any"
		if analysis.paramMask&(uint32(1)<<index) != 0 {
			typ = "int64"
			paramTypes[index] = compiler.IRInt
		}
		typedSignature[index] = typedParams[index] + " " + typ
	}
	g.writef("%s = func(%s) any {\n",
		target.int64ParamFnVar,
		strings.Join(typedSignature, ", "),
	)
	plainIR := g.currentIR
	g.currentIR = analysis.ir
	g.generateFnMethodFixedWithTypes(method, typedParams, paramTypes)
	g.currentIR = plainIR
	g.writef("}\n")

	signature := ""
	if method.FixedArity > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, method.FixedArity, signature)

	typedNames := append([]string(nil), paramNames...)
	guards := make([]string, 0, bits.OnesCount32(analysis.paramMask))
	for i, paramName := range paramNames {
		if analysis.paramMask&(uint32(1)<<i) == 0 {
			continue
		}
		typed := g.allocateTempVar()
		ok := g.allocateTempVar()
		g.writef("%s, %s := %s.(int64)\n", typed, ok, paramName)
		typedNames[i] = typed
		guards = append(guards, ok)
	}
	if len(guards) == 0 {
		panic(fmt.Sprintf("empty int64 parameter guard for %v", target.vr))
	}
	g.writef("if %s {\n", strings.Join(guards, " && "))
	g.writef("return %s(%s)\n",
		target.int64ParamFnVar,
		strings.Join(typedNames, ", "),
	)
	g.writef("}\n")
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (g *Generator) generateAOTInt64ParameterInvoke(
	node *ast.Node,
) (string, bool) {
	if !g.directLink || g.currentIR == nil ||
		node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	target := g.aotInvokeTarget(invoke)
	if target == nil || !target.directLinked ||
		target.int64ParamAnalysis == nil ||
		len(invoke.Args) != target.arity {
		return "", false
	}
	mask := target.int64ParamAnalysis.paramMask
	for index, argument := range invoke.Args {
		if mask&(uint32(1)<<index) != 0 &&
			(g.currentIR.Facts(argument).Type.Kind != compiler.IRInt ||
				g.currentIR.Facts(argument).Type.GoType !=
					reflect.TypeOf(int64(0))) {
			return "", false
		}
	}
	args := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		code := g.generateASTNode(argument)
		args[index] = code
		if mask&(uint32(1)<<index) != 0 {
			args[index] = g.irInt64Expr(argument, code)
		}
	}
	result := g.allocateTempVar()
	g.writef("// direct int64 parameter call %s\n", target.vr)
	g.writef("%s := %s(%s)\n",
		result,
		target.int64ParamFnVar,
		strings.Join(args, ", "),
	)
	return result, true
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
