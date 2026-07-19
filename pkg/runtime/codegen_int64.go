//go:build !glj_aot_runtime

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

type int64AOTAnalysis struct {
	target             *aotSpecializationTarget
	arity              int
	usesSelf           bool
	uncheckedHostCalls map[*ast.HostCallNode]bool
	guardInt32Params   bool
	guardInt32Loops    map[*ast.LetNode]bool
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
		target:             target,
		arity:              method.FixedArity,
		uncheckedHostCalls: make(map[*ast.HostCallNode]bool),
	}
	analyzer := newInt64AOTAnalyzer(analysis, targets)
	locals := make(map[*lang.Symbol]aotPrimitiveType, method.FixedArity)
	for _, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = int64AOTPrimitive
	}
	if analyzer.exprType(method.Body, locals) != int64AOTPrimitive {
		return nil
	}
	return analysis
}

func newInt64AOTAnalyzer(
	analysis *int64AOTAnalysis,
	targets map[*lang.Var]*aotSpecializationTarget,
) *primitiveAOTAnalyzer {
	return &primitiveAOTAnalyzer{
		target:     analysis.target,
		arity:      analysis.arity,
		paramType:  int64AOTPrimitive,
		resultType: int64AOTPrimitive,
		targets:    targets,
		markUsesSelf: func() {
			analysis.usesSelf = true
		},
		hasTypedTarget: func(target *aotSpecializationTarget) bool {
			return target.int64Analysis != nil
		},
	}
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
	if analysis.guardInt32Params {
		g.writeInt32AOTFallbackGuards(typedParams)
	}
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
		typ := newInt64AOTAnalyzer(
			e.analysis,
			e.g.aotCallTargets,
		).exprType(node, aotLocalTypes(locals))
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
	analyzer := newInt64AOTAnalyzer(e.analysis, e.g.aotCallTargets)
	typ := analyzer.exprType(letNode.Body, aotLocalTypesAfterBindings(
		analyzer, letNode.Bindings, aotLocalTypes(locals),
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
	if e.analysis.guardInt32Loops[loop] {
		names := make([]string, len(bindings))
		for i, binding := range bindings {
			names[i] = binding.name
		}
		e.g.writeInt32AOTFallbackGuards(names)
	}
	e.emitLoopTail(loop.Body, nested, loop.LoopID, bindings, result, label)
	e.g.writef("}\n")
	e.g.writef("}\n")
	return result
}

func (g *Generator) writeInt32AOTFallbackGuards(names []string) {
	for _, name := range names {
		g.writef("if %s < -2147483647 || %s > 2147483647 {\n",
			name, name)
		g.writef("return 0, false\n")
		g.writef("}\n")
	}
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
		analyzer := newInt64AOTAnalyzer(e.analysis, e.g.aotCallTargets)
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
	unchecked := e.analysis.uncheckedHostCalls[call]
	if len(args) == 1 {
		switch name {
		case "inc":
			if unchecked {
				return "(" + args[0] + " + 1)"
			}
			return "lang.CheckedAddInt64(" + args[0] + ", 1)"
		case "unchecked_inc":
			return "(" + args[0] + " + 1)"
		case "dec":
			if unchecked {
				return "(" + args[0] + " - 1)"
			}
			return "lang.CheckedSubInt64(" + args[0] + ", 1)"
		case "uncheckeddec", "unchecked_dec":
			return "(" + args[0] + " - 1)"
		case "minus":
			if unchecked {
				return "(-" + args[0] + ")"
			}
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
		if unchecked {
			return "(" + args[0] + " + " + args[1] + ")"
		}
		return "lang.CheckedAddInt64(" + args[0] + ", " + args[1] + ")"
	case "uncheckedadd":
		return "(" + args[0] + " + " + args[1] + ")"
	case "minus":
		if unchecked {
			return "(" + args[0] + " - " + args[1] + ")"
		}
		return "lang.CheckedSubInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_minus":
		return "(" + args[0] + " - " + args[1] + ")"
	case "multiply":
		if unchecked {
			return "(" + args[0] + " * " + args[1] + ")"
		}
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
	analyzer *primitiveAOTAnalyzer,
	bindings []*ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
) map[*lang.Symbol]aotPrimitiveType {
	for _, binding := range bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		locals[bindingNode.Name] = analyzer.exprType(bindingNode.Init, locals)
	}
	return locals
}
