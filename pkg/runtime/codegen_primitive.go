//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type aotPrimitiveType uint8

const (
	invalidAOTPrimitive aotPrimitiveType = iota
	int64AOTPrimitive
	float64AOTPrimitive
	boolAOTPrimitive
)

// primitiveAOTAnalyzer owns the structural type propagation shared by typed
// AOT variants. Each variant supplies its parameter/result type and target
// availability, while lets, loops, branches, numeric promotion, and direct
// calls follow one set of rules.
type primitiveAOTAnalyzer struct {
	target         *aotSpecializationTarget
	arity          int
	paramType      aotPrimitiveType
	resultType     aotPrimitiveType
	allowFloat     bool
	allowCoreMod   bool
	targets        map[*lang.Var]*aotSpecializationTarget
	markUsesSelf   func()
	markCallee     func(*aotSpecializationTarget)
	hasTypedTarget func(*aotSpecializationTarget) bool
}

func (a *primitiveAOTAnalyzer) exprType(
	node *ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		var typ aotPrimitiveType
		switch node.Sub.(*ast.ConstNode).Value.(type) {
		case int64:
			typ = int64AOTPrimitive
		case float64:
			typ = float64AOTPrimitive
		case bool:
			typ = boolAOTPrimitive
		}
		if a.accepts(typ) {
			return typ
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
		bindingTypes := make([]aotPrimitiveType, len(loop.Bindings))
		for i, binding := range loop.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			typ := a.exprType(bindingNode.Init, nested)
			if !a.acceptsNumeric(typ) {
				return invalidAOTPrimitive
			}
			bindingTypes[i] = typ
			nested[bindingNode.Name] = typ
		}
		if a.loopTail(
			loop.Body,
			nested,
			loop.LoopID,
			bindingTypes,
			a.resultType,
		) {
			return a.resultType
		}

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		if a.exprType(ifNode.Test, locals) != boolAOTPrimitive {
			return invalidAOTPrimitive
		}
		thenType := a.exprType(ifNode.Then, locals)
		if thenType != invalidAOTPrimitive &&
			a.exprType(ifNode.Else, locals) == thenType {
			return thenType
		}

	case ast.OpHostCall:
		return a.hostCallType(node.Sub.(*ast.HostCallNode), locals)

	case ast.OpInvoke:
		return a.invokeType(node.Sub.(*ast.InvokeNode), locals)
	}
	return invalidAOTPrimitive
}

func (a *primitiveAOTAnalyzer) loopTail(
	node *ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
	loopID *lang.Symbol,
	bindingTypes []aotPrimitiveType,
	resultType aotPrimitiveType,
) bool {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpRecur:
		recur := node.Sub.(*ast.RecurNode)
		if !lang.Equals(recur.LoopID, loopID) ||
			len(recur.Exprs) != len(bindingTypes) {
			return false
		}
		for i, expr := range recur.Exprs {
			if a.exprType(expr, locals) != bindingTypes[i] {
				return false
			}
		}
		return true

	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		return a.exprType(ifNode.Test, locals) == boolAOTPrimitive &&
			a.loopTail(ifNode.Then, locals, loopID, bindingTypes, resultType) &&
			a.loopTail(ifNode.Else, locals, loopID, bindingTypes, resultType)

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
		return a.loopTail(
			letNode.Body,
			nested,
			loopID,
			bindingTypes,
			resultType,
		)
	}
	return a.exprType(node, locals) == resultType
}

func (a *primitiveAOTAnalyzer) hostCallType(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	if !isNumbersCall(call) {
		return invalidAOTPrimitive
	}
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		typ := a.exprType(call.Args[0], locals)
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec",
			"minus", "unchecked_minus":
			if a.acceptsNumeric(typ) {
				return typ
			}
		case "iszero", "ispos", "isneg":
			if a.acceptsNumeric(typ) {
				return boolAOTPrimitive
			}
		}
		return invalidAOTPrimitive
	}
	if len(call.Args) != 2 {
		return invalidAOTPrimitive
	}
	left := a.exprType(call.Args[0], locals)
	right := a.exprType(call.Args[1], locals)
	if !a.acceptsNumeric(left) || !a.acceptsNumeric(right) {
		return invalidAOTPrimitive
	}
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply":
		return promotedAOTNumericType(left, right)
	case "divide":
		if left == float64AOTPrimitive || right == float64AOTPrimitive {
			return float64AOTPrimitive
		}
	case "quotient", "remainder":
		if left == int64AOTPrimitive && right == int64AOTPrimitive {
			return int64AOTPrimitive
		}
	case "lt", "lte", "gt", "gte", "equiv":
		// Clojure preserves more precision than an int64-to-float64
		// conversion for mixed numeric comparisons.
		if left == right {
			return boolAOTPrimitive
		}
	}
	return invalidAOTPrimitive
}

func (a *primitiveAOTAnalyzer) invokeType(
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]aotPrimitiveType,
) aotPrimitiveType {
	if invoke.Fn.Op != ast.OpVar {
		return invalidAOTPrimitive
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if a.target != nil && vr == a.target.vr &&
		a.allType(invoke.Args, locals, a.arity, a.paramType) {
		a.markUsesSelf()
		return a.resultType
	}
	if target := a.targets[vr]; target != nil &&
		a.hasTypedTarget(target) &&
		a.allType(invoke.Args, locals, target.arity, a.paramType) {
		if a.markCallee != nil {
			a.markCallee(target)
		}
		return a.resultType
	}
	if vr.String() == "#'clojure.core/=" && len(invoke.Args) == 2 {
		left := a.exprType(invoke.Args[0], locals)
		right := a.exprType(invoke.Args[1], locals)
		if a.acceptsNumeric(left) && left == right {
			return boolAOTPrimitive
		}
	}
	if a.allowCoreMod && vr.Namespace() != nil &&
		vr.Namespace().Name().String() == "clojure.core" &&
		vr.Symbol().String() == "mod" &&
		!vr.IsDynamic() &&
		!RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) &&
		IsDefaultCoreVar(vr) &&
		a.allType(invoke.Args, locals, 2, int64AOTPrimitive) {
		return int64AOTPrimitive
	}
	return invalidAOTPrimitive
}

func (a *primitiveAOTAnalyzer) allType(
	args []*ast.Node,
	locals map[*lang.Symbol]aotPrimitiveType,
	arity int,
	typ aotPrimitiveType,
) bool {
	if len(args) != arity {
		return false
	}
	for _, arg := range args {
		if a.exprType(arg, locals) != typ {
			return false
		}
	}
	return true
}

func (a *primitiveAOTAnalyzer) accepts(typ aotPrimitiveType) bool {
	return typ == boolAOTPrimitive || a.acceptsNumeric(typ)
}

func (a *primitiveAOTAnalyzer) acceptsNumeric(typ aotPrimitiveType) bool {
	return typ == int64AOTPrimitive ||
		a.allowFloat && typ == float64AOTPrimitive
}

func promotedAOTNumericType(left, right aotPrimitiveType) aotPrimitiveType {
	if left == float64AOTPrimitive || right == float64AOTPrimitive {
		return float64AOTPrimitive
	}
	return int64AOTPrimitive
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

func aotGoType(typ aotPrimitiveType) string {
	switch typ {
	case float64AOTPrimitive:
		return "float64"
	case boolAOTPrimitive:
		return "bool"
	default:
		return "int64"
	}
}
