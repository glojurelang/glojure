package compiler

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// analyzeOwnedStringParts proves that a loop-carried empty vector is used
// only as an append-only collection of arguments for clojure.core/str. The
// elements remain unconverted until the apply boundary, preserving the order
// and timing of ToString calls.
func (ir *TypedIR) analyzeOwnedStringParts(
	binding *ast.Node,
	bindingIndex int,
	loop *ast.LetNode,
) bool {
	bindingNode := binding.Sub.(*ast.BindingNode)
	if !irEmptyVectorLiteral(bindingNode.Init) || loop.LoopID == nil {
		return false
	}

	usage := ownedStringPartsUsage{
		target:       bindingNode.Name,
		bindingIndex: bindingIndex,
		loopID:       loop.LoopID,
		allowed:      make(map[*ast.Node]bool),
		safe:         true,
	}
	usage.scan(loop.Body)
	if !usage.safe || len(usage.appends) == 0 ||
		len(usage.finishes) == 0 {
		return false
	}
	usage.rejectOtherReferences(loop.Body)
	if !usage.safe {
		return false
	}

	for _, appendNode := range usage.appends {
		invoke := appendNode.Sub.(*ast.InvokeNode)
		facts := ir.facts[appendNode]
		facts.StringPartsAppend = &IROwnedStringPartsAppendPlan{
			Parts: bindingNode.Name,
			Value: invoke.Args[1],
		}
		facts.Type = IRType{Kind: IRVector}
		ir.facts[appendNode] = facts
	}
	for _, finishNode := range usage.finishes {
		facts := ir.facts[finishNode]
		facts.StringPartsFinish = &IROwnedStringPartsFinishPlan{
			Parts: bindingNode.Name,
		}
		facts.Type = IRType{Kind: IRString}
		ir.facts[finishNode] = facts
	}
	return true
}

type ownedStringPartsUsage struct {
	target       *lang.Symbol
	bindingIndex int
	loopID       *lang.Symbol
	allowed      map[*ast.Node]bool
	appends      []*ast.Node
	finishes     []*ast.Node
	safe         bool
}

func (u *ownedStringPartsUsage) scan(node *ast.Node) {
	if node == nil || !u.safe {
		return
	}
	if node.Op == ast.OpRecur {
		recur := node.Sub.(*ast.RecurNode)
		if recur.LoopID == u.loopID {
			if u.bindingIndex >= len(recur.Exprs) {
				u.safe = false
				return
			}
			appendNode := recur.Exprs[u.bindingIndex]
			local, ok := matchOwnedStringPartsAppend(
				appendNode,
				u.target,
			)
			if !ok {
				u.safe = false
				return
			}
			u.allowed[local] = true
			u.appends = append(u.appends, appendNode)
		}
	}
	if local, ok := matchOwnedStringPartsFinish(node, u.target); ok {
		u.allowed[local] = true
		u.finishes = append(u.finishes, node)
	}
	for _, child := range irChildren(node) {
		u.scan(child)
	}
}

func (u *ownedStringPartsUsage) rejectOtherReferences(node *ast.Node) {
	if node == nil || !u.safe {
		return
	}
	if irLocalIs(node, u.target) && !u.allowed[node] {
		u.safe = false
		return
	}
	for _, child := range irChildren(node) {
		u.rejectOtherReferences(child)
	}
}

func matchOwnedStringPartsAppend(
	node *ast.Node,
	target *lang.Symbol,
) (*ast.Node, bool) {
	if node == nil || node.Op != ast.OpInvoke {
		return nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "conj" || len(invoke.Args) != 2 ||
		!irLocalIs(invoke.Args[0], target) {
		return nil, false
	}
	return invoke.Args[0], true
}

func matchOwnedStringPartsFinish(
	node *ast.Node,
	target *lang.Symbol,
) (*ast.Node, bool) {
	if node == nil || node.Op != ast.OpInvoke {
		return nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "apply" || len(invoke.Args) != 2 ||
		!irLocalIs(invoke.Args[1], target) ||
		invoke.Args[0] == nil || invoke.Args[0].Op != ast.OpVar {
		return nil, false
	}
	strVar := invoke.Args[0].Sub.(*ast.VarNode).Var
	if strVar == nil || strVar.Namespace() == nil ||
		strVar.Namespace().Name().String() != "clojure.core" ||
		strVar.Symbol().String() != "str" {
		return nil, false
	}
	return invoke.Args[1], true
}

func irEmptyVectorLiteral(node *ast.Node) bool {
	return node != nil && node.Op == ast.OpVector &&
		len(node.Sub.(*ast.VectorNode).Items) == 0
}
