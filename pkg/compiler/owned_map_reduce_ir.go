package compiler

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// analyzeOwnedMapReduce proves that an inline two-argument reducer threads an
// empty map literal through update-in calls without exposing an intermediate
// accumulator. Keeping the initial form narrow gives the runtime lowering a
// representation it can always convert to a transient without a speculative
// fallback.
func (ir *TypedIR) analyzeOwnedMapReduce(
	node *ast.Node,
) *IROwnedMapReducePlan {
	if node == nil || node.Op != ast.OpInvoke {
		return nil
	}
	invoke := node.Sub.(*ast.InvokeNode)
	reduceVar, name, core := irCoreCall(invoke)
	if !core || name != "reduce" || len(invoke.Args) != 3 ||
		!irEmptyMapLiteral(invoke.Args[1]) ||
		!irFixedFnArity(invoke.Args[0], 2) {
		return nil
	}

	fn := invoke.Args[0].Sub.(*ast.FnNode)
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	if len(method.Params) != 2 {
		return nil
	}
	accumulator := method.Params[0].Sub.(*ast.BindingNode).Name
	usage := ownedMapReduceUsage{
		accumulator: accumulator,
		allowed:     make(map[*ast.Node]bool),
		safe:        true,
	}
	usage.scanTail(method.Body)
	if !usage.safe || len(usage.updates) == 0 {
		return nil
	}
	usage.rejectOtherReferences(method.Body)
	if !usage.safe {
		return nil
	}

	updateVars := make([]*lang.Var, 0, len(usage.updates))
	seen := make(map[*lang.Var]bool)
	for _, update := range usage.updates {
		facts := ir.facts[update]
		updatePlan := &IROwnedMapUpdatePlan{}
		invoke := update.Sub.(*ast.InvokeNode)
		if path := invoke.Args[1]; path.Op == ast.OpVector {
			updatePlan.Keys = append(
				[]*ast.Node(nil),
				path.Sub.(*ast.VectorNode).Items...,
			)
		}
		if callback := invoke.Args[2]; callback.Op == ast.OpInvoke {
			callbackInvoke := callback.Sub.(*ast.InvokeNode)
			if vr, name, core := irCoreCall(callbackInvoke); core &&
				name == "fnil" && len(callbackInvoke.Args) == 2 {
				updatePlan.Fnil = &IROwnedMapFnilPlan{
					Var:     vr,
					Fn:      callbackInvoke.Args[0],
					Default: callbackInvoke.Args[1],
				}
			}
		}
		facts.OwnedMapUpdateIn = updatePlan
		ir.facts[update] = facts
		vr, _, _ := irCoreCall(invoke)
		if !seen[vr] {
			seen[vr] = true
			updateVars = append(updateVars, vr)
		}
	}
	return &IROwnedMapReducePlan{
		ReduceVar:    reduceVar,
		UpdateInVars: updateVars,
		Reducer:      invoke.Args[0],
		Initial:      invoke.Args[1],
		Source:       invoke.Args[2],
		Updates:      append([]*ast.Node(nil), usage.updates...),
	}
}

type ownedMapReduceUsage struct {
	accumulator *lang.Symbol
	allowed     map[*ast.Node]bool
	updates     []*ast.Node
	safe        bool
}

func (u *ownedMapReduceUsage) scanTail(node *ast.Node) {
	node = irUnwrapDo(node)
	if node == nil || !u.safe {
		u.safe = false
		return
	}
	switch node.Op {
	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		for _, binding := range let.Bindings {
			if u.containsAccumulator(
				binding.Sub.(*ast.BindingNode).Init,
			) {
				u.safe = false
				return
			}
		}
		u.scanTail(let.Body)
	case ast.OpDo:
		do := node.Sub.(*ast.DoNode)
		for _, statement := range do.Statements {
			if u.containsAccumulator(statement) {
				u.safe = false
				return
			}
		}
		u.scanTail(do.Ret)
	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		if u.containsAccumulator(conditional.Test) {
			u.safe = false
			return
		}
		u.scanTail(conditional.Then)
		u.scanTail(conditional.Else)
	default:
		if !u.scanUpdateChain(node) {
			u.safe = false
		}
	}
}

func (u *ownedMapReduceUsage) scanUpdateChain(node *ast.Node) bool {
	node = irUnwrapDo(node)
	if irLocalIs(node, u.accumulator) {
		u.allowed[node] = true
		return true
	}
	if node == nil || node.Op != ast.OpInvoke {
		return false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "update-in" || len(invoke.Args) < 3 ||
		!u.scanUpdateChain(invoke.Args[0]) {
		return false
	}
	if u.containsAccumulator(invoke.Fn) {
		return false
	}
	for _, argument := range invoke.Args[1:] {
		if u.containsAccumulator(argument) {
			return false
		}
	}
	u.updates = append(u.updates, node)
	return true
}

func (u *ownedMapReduceUsage) rejectOtherReferences(node *ast.Node) {
	if node == nil || !u.safe {
		return
	}
	if irLocalIs(node, u.accumulator) && !u.allowed[node] {
		u.safe = false
		return
	}
	for _, child := range irChildren(node) {
		u.rejectOtherReferences(child)
	}
}

func (u *ownedMapReduceUsage) containsAccumulator(node *ast.Node) bool {
	if node == nil {
		return false
	}
	if irLocalIs(node, u.accumulator) {
		return true
	}
	for _, child := range irChildren(node) {
		if u.containsAccumulator(child) {
			return true
		}
	}
	return false
}

func irEmptyMapLiteral(node *ast.Node) bool {
	if node == nil || node.Op != ast.OpMap {
		return false
	}
	m := node.Sub.(*ast.MapNode)
	return len(m.Keys) == 0 && len(m.Vals) == 0
}
