//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// float64 AOT specialization complements the integer specialization for
// functions whose complete result is floating point. Parameters are guarded
// at the boxed entry point, while integer constants and loop bindings retain
// their integer type inside otherwise floating-point regions. The ordinary
// generated function remains the fallback for non-float arguments and
// redefined Vars.
type float64AOTAnalysis struct {
	target   *aotSpecializationTarget
	arity    int
	usesSelf bool
}

func analyzeFloat64AOTFunction(
	target *aotSpecializationTarget,
	method *ast.FnMethodNode,
	targets map[*lang.Var]*aotSpecializationTarget,
) *float64AOTAnalysis {
	if target == nil || method.IsVariadic || method.FixedArity > 4 {
		return nil
	}
	analysis := &float64AOTAnalysis{
		target: target,
		arity:  method.FixedArity,
	}
	analyzer := newFloat64AOTAnalyzer(analysis, targets)
	locals := make(map[*lang.Symbol]aotPrimitiveType, method.FixedArity)
	for _, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = float64AOTPrimitive
	}
	if analyzer.exprType(method.Body, locals) != float64AOTPrimitive {
		return nil
	}
	return analysis
}

func newFloat64AOTAnalyzer(
	analysis *float64AOTAnalysis,
	targets map[*lang.Var]*aotSpecializationTarget,
) *primitiveAOTAnalyzer {
	return &primitiveAOTAnalyzer{
		target:     analysis.target,
		arity:      analysis.arity,
		paramType:  float64AOTPrimitive,
		resultType: float64AOTPrimitive,
		allowFloat: true,
		targets:    targets,
		markUsesSelf: func() {
			analysis.usesSelf = true
		},
		hasTypedTarget: func(target *aotSpecializationTarget) bool {
			return target.float64Analysis != nil
		},
	}
}

type float64AOTEmitter struct {
	g        *Generator
	analysis *float64AOTAnalysis
	helper   string
}

func (g *Generator) generateFloat64SpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn {
		return false
	}
	analysis := target.float64Analysis
	if analysis == nil {
		return false
	}

	helper := g.allocateTempVar()
	typedParams := make([]string, method.FixedArity)
	typedSignature := make([]string, method.FixedArity)
	locals := make(map[*lang.Symbol]aotTypedLocal, method.FixedArity)
	for i, param := range method.Params {
		typedParams[i] = g.allocateTempVar()
		typedSignature[i] = typedParams[i] + " float64"
		locals[param.Sub.(*ast.BindingNode).Name] = aotTypedLocal{
			name: typedParams[i],
			typ:  float64AOTPrimitive,
		}
	}

	g.writef("var %s func(%s) (float64, bool)\n",
		helper, strings.Join(typedSignature, ", "))
	g.writef("%s = func(%s) (float64, bool) {\n",
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
	emitter := float64AOTEmitter{g: g, analysis: analysis, helper: helper}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s, true\n", result)
	g.writef("}\n")
	g.writef("%s = %s\n", target.float64FnVar, helper)

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
		g.writef("%s, %s := %s.(float64)\n",
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

func (e *float64AOTEmitter) emitExpr(
	node *ast.Node,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		switch value := node.Sub.(*ast.ConstNode).Value.(type) {
		case int64:
			return "int64(" + strconv.FormatInt(value, 10) + ")"
		case float64:
			return "float64(" + strconv.FormatFloat(value, 'g', -1, 64) + ")"
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
		analyzer := e.analyzer()
		typ := analyzer.exprType(node, aotLocalTypes(locals))
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
	panic(fmt.Sprintf("unsupported float64 AOT expression: %v", node.Op))
}

func (e *float64AOTEmitter) emitLet(
	letNode *ast.LetNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	analyzer := e.analyzer()
	types := aotLocalTypes(locals)
	for _, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		types[bindingNode.Name] = analyzer.exprType(bindingNode.Init, types)
	}
	typ := analyzer.exprType(letNode.Body, types)
	result := e.g.allocateTempVar()
	e.g.writef("var %s %s\n", result, aotGoType(typ))
	e.g.writef("{\n")
	nested := cloneAOTLocals(locals)
	for _, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		valueType := analyzer.exprType(bindingNode.Init, aotLocalTypes(nested))
		value := e.emitExpr(bindingNode.Init, nested)
		name := e.g.allocateTempVar()
		e.g.writef("%s := %s\n", name, value)
		nested[bindingNode.Name] = aotTypedLocal{name: name, typ: valueType}
	}
	body := e.emitExpr(letNode.Body, nested)
	e.g.writef("%s = %s\n", result, body)
	e.g.writef("}\n")
	return result
}

func (e *float64AOTEmitter) emitLoop(
	loop *ast.LetNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	result := e.g.allocateTempVar()
	label := "float64_loop_" + e.g.allocateTempVar()
	e.g.writef("var %s float64\n", result)
	e.g.writef("{\n")
	nested := cloneAOTLocals(locals)
	bindings := make([]aotTypedLocal, len(loop.Bindings))
	analyzer := e.analyzer()
	for i, binding := range loop.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		typ := analyzer.exprType(bindingNode.Init, aotLocalTypes(nested))
		value := e.emitExpr(bindingNode.Init, nested)
		name := e.g.allocateTempVar()
		e.g.writef("%s := %s\n", name, value)
		local := aotTypedLocal{name: name, typ: typ}
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

func (e *float64AOTEmitter) emitLoopTail(
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
		analyzer := e.analyzer()
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			typ := analyzer.exprType(bindingNode.Init, aotLocalTypes(nested))
			value := e.emitExpr(bindingNode.Init, nested)
			name := e.g.allocateTempVar()
			e.g.writef("%s := %s\n", name, value)
			nested[bindingNode.Name] = aotTypedLocal{name: name, typ: typ}
		}
		e.emitLoopTail(letNode.Body, nested, loopID, bindings, result, label)
		e.g.writef("}\n")

	default:
		value := e.emitExpr(node, locals)
		e.g.writef("%s = %s\n", result, value)
		e.g.writef("break %s\n", label)
	}
}

func (e *float64AOTEmitter) emitHostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	analyzer := e.analyzer()
	localTypes := aotLocalTypes(locals)
	args := make([]string, len(call.Args))
	types := make([]aotPrimitiveType, len(call.Args))
	for i, arg := range call.Args {
		args[i] = e.emitExpr(arg, locals)
		types[i] = analyzer.exprType(arg, localTypes)
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc", "unchecked_inc":
			if types[0] == float64AOTPrimitive {
				return "(" + args[0] + " + 1)"
			}
			if name == "unchecked_inc" {
				return "(" + args[0] + " + 1)"
			}
			return "lang.CheckedAddInt64(" + args[0] + ", 1)"
		case "dec", "uncheckeddec", "unchecked_dec":
			if types[0] == float64AOTPrimitive {
				return "(" + args[0] + " - 1)"
			}
			if name != "dec" {
				return "(" + args[0] + " - 1)"
			}
			return "lang.CheckedSubInt64(" + args[0] + ", 1)"
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
			return "lang.CheckedMultiplyInt64(" + args[0] + ", " + args[1] + ")"
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
	panic("unsupported float64 AOT host call")
}

func (e *float64AOTEmitter) emitInvoke(
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	analyzer := e.analyzer()
	localTypes := aotLocalTypes(locals)
	args := make([]string, len(invoke.Args))
	types := make([]aotPrimitiveType, len(invoke.Args))
	for i, arg := range invoke.Args {
		args[i] = e.emitExpr(arg, locals)
		types[i] = analyzer.exprType(arg, localTypes)
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.String() == "#'clojure.core/=" {
		if types[0] != types[1] {
			if types[0] == int64AOTPrimitive {
				args[0] = "float64(" + args[0] + ")"
			}
			if types[1] == int64AOTPrimitive {
				args[1] = "float64(" + args[1] + ")"
			}
		}
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
		helper = target.float64FnVar
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

func (e *float64AOTEmitter) analyzer() *primitiveAOTAnalyzer {
	return newFloat64AOTAnalyzer(e.analysis, e.g.aotCallTargets)
}
