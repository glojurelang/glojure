package compiler

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// analyzeOwnedMapReduce proves that an inline two-argument reducer linearly
// threads a map through ownership-compatible mutations without exposing an
// intermediate accumulator. Eligibility follows from representation and
// escape facts; the particular keys, paths, callbacks, and number of
// mutations do not affect the proof.
func (ir *TypedIR) analyzeOwnedMapReduce(
	node *ast.Node,
) *IROwnedMapReducePlan {
	if node == nil || node.Op != ast.OpInvoke {
		return nil
	}
	invoke := node.Sub.(*ast.InvokeNode)
	reduceVar, name, core := irCoreCall(invoke)
	if !core || name != "reduce" || len(invoke.Args) != 3 ||
		invoke.Args[1].Op != ast.OpMap ||
		!irFixedFnArity(invoke.Args[0], 2) {
		return nil
	}

	fn := invoke.Args[0].Sub.(*ast.FnNode)
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	if len(method.Params) != 2 {
		return nil
	}
	accumulator := method.Params[0].Sub.(*ast.BindingNode).Name
	if irNestedFunctionCapturesLocal(method.Body, accumulator) {
		return nil
	}
	usage := ownedMapReduceUsage{
		accumulator: accumulator,
		allowed:     make(map[*ast.Node]bool),
		safe:        true,
	}
	usage.scanTail(method.Body)
	if !usage.safe || len(usage.mutations) == 0 {
		return nil
	}
	usage.rejectOtherReferences(method.Body)
	if !usage.safe {
		return nil
	}

	callVars := make([]*lang.Var, 0, len(usage.mutations))
	seen := make(map[*lang.Var]bool)
	for _, mutation := range usage.mutations {
		facts := ir.facts[mutation]
		plan := ownedMapMutationPlan(mutation)
		facts.OwnedMapMutation = plan
		ir.facts[mutation] = facts
		if mutation.Op != ast.OpInvoke {
			continue
		}
		vr, _, _ := irCoreCall(mutation.Sub.(*ast.InvokeNode))
		if vr != nil && !seen[vr] {
			seen[vr] = true
			callVars = append(callVars, vr)
		}
	}
	return &IROwnedMapReducePlan{
		ReduceVar: reduceVar,
		CallVars:  callVars,
		Reducer:   invoke.Args[0],
		Initial:   invoke.Args[1],
		Source:    invoke.Args[2],
		Mutations: append([]*ast.Node(nil), usage.mutations...),
	}
}

func ownedMapMutationPlan(node *ast.Node) *IROwnedMapMutationPlan {
	if node.Op == ast.OpAssoc {
		return &IROwnedMapMutationPlan{Kind: IROwnedMapAssoc}
	}
	plan := &IROwnedMapMutationPlan{Kind: IROwnedMapUpdateIn}
	invoke := node.Sub.(*ast.InvokeNode)
	if path := invoke.Args[1]; path.Op == ast.OpVector {
		plan.Keys = append(
			[]*ast.Node(nil), path.Sub.(*ast.VectorNode).Items...,
		)
	}
	if callback := invoke.Args[2]; callback.Op == ast.OpInvoke {
		callbackInvoke := callback.Sub.(*ast.InvokeNode)
		if vr, name, core := irCoreCall(callbackInvoke); core &&
			name == "fnil" && len(callbackInvoke.Args) == 2 {
			plan.Fnil = &IROwnedMapFnilPlan{
				Var:     vr,
				Fn:      callbackInvoke.Args[0],
				Default: callbackInvoke.Args[1],
			}
		}
	}
	return plan
}

type ownedMapReduceUsage struct {
	accumulator *lang.Symbol
	allowed     map[*ast.Node]bool
	mutations   []*ast.Node
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
		if !u.scanMutationChain(node) {
			u.safe = false
		}
	}
}

func (u *ownedMapReduceUsage) scanMutationChain(node *ast.Node) bool {
	node = irUnwrapDo(node)
	if irLocalIs(node, u.accumulator) {
		u.allowed[node] = true
		return true
	}
	if node == nil {
		return false
	}
	if node.Op == ast.OpAssoc {
		assoc := node.Sub.(*ast.AssocNode)
		if len(assoc.Entries) == 0 ||
			!u.scanMutationChain(assoc.Target) {
			return false
		}
		for _, entry := range assoc.Entries {
			if u.containsAccumulator(entry.Key) ||
				u.containsAccumulator(entry.Val) {
				return false
			}
		}
		u.mutations = append(u.mutations, node)
		return true
	}
	if node.Op != ast.OpInvoke {
		return false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "update-in" || len(invoke.Args) < 3 ||
		!u.scanMutationChain(invoke.Args[0]) {
		return false
	}
	// update-in with an empty path invokes the callback on the accumulator
	// itself. A dynamic path might be empty, which would expose the private
	// mutable representation to arbitrary user code. Require a literal,
	// proven-nonempty path before treating the operation as an owned mutation.
	path := invoke.Args[1]
	if path == nil || path.Op != ast.OpVector ||
		len(path.Sub.(*ast.VectorNode).Items) == 0 {
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
	u.mutations = append(u.mutations, node)
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
