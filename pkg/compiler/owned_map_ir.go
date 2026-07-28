package compiler

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// analyzeOwnedMap proves that a loop-carried persistent map has one lexical
// owner. The map may be observed through non-escaping lookup operations, may
// be replaced by assoc in its own recur slot, and may leave the loop only as a
// terminal result. These constraints make a transient representation
// observationally equivalent to the persistent values it replaces.
func (ir *TypedIR) analyzeOwnedMap(
	binding *ast.Node,
	bindingIndex int,
	loop *ast.LetNode,
) bool {
	bindingNode := binding.Sub.(*ast.BindingNode)
	if ir.facts[bindingNode.Init].Type.Kind != IRMap {
		return false
	}
	usage := ownedMapUsage{
		ir:           ir,
		target:       bindingNode.Name,
		bindingIndex: bindingIndex,
		loopID:       loop.LoopID,
		safe:         true,
	}
	usage.scanTail(loop.Body)
	if !usage.safe || usage.updates == 0 || usage.exits == 0 {
		return false
	}
	for _, update := range usage.updateNodes {
		facts := ir.facts[update]
		facts.OwnedMapAssoc = true
		ir.facts[update] = facts
	}
	for _, exit := range usage.exitNodes {
		facts := ir.facts[exit]
		facts.PersistOwnedMap = true
		ir.facts[exit] = facts
	}
	return true
}

type ownedMapUsage struct {
	ir           *TypedIR
	target       *lang.Symbol
	bindingIndex int
	loopID       *lang.Symbol
	safe         bool
	updates      int
	exits        int
	updateNodes  []*ast.Node
	exitNodes    []*ast.Node
}

// scanTail follows expression-result positions through control flow. A recur
// keeps ownership inside the loop; an exact local result freezes the map.
func (u *ownedMapUsage) scanTail(node *ast.Node) {
	node = irUnwrapDo(node)
	if node == nil || !u.safe {
		return
	}
	switch node.Op {
	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		u.scanRead(conditional.Test)
		u.scanTail(conditional.Then)
		u.scanTail(conditional.Else)
	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		for _, binding := range let.Bindings {
			u.scanRead(binding.Sub.(*ast.BindingNode).Init)
		}
		u.scanTail(let.Body)
	case ast.OpDo:
		do := node.Sub.(*ast.DoNode)
		for _, statement := range do.Statements {
			u.scanRead(statement)
		}
		u.scanTail(do.Ret)
	case ast.OpRecur:
		u.scanRecur(node)
	case ast.OpLocal:
		if irLocalIs(node, u.target) {
			u.exits++
			u.exitNodes = append(u.exitNodes, node)
		}
	default:
		u.scanRead(node)
	}
}

func (u *ownedMapUsage) scanRecur(node *ast.Node) {
	recur := node.Sub.(*ast.RecurNode)
	if recur.LoopID != u.loopID ||
		u.bindingIndex >= len(recur.Exprs) {
		u.scanRead(node)
		return
	}
	for index, expression := range recur.Exprs {
		if index != u.bindingIndex {
			u.scanRead(expression)
			continue
		}
		if expression.Op != ast.OpAssoc {
			u.safe = false
			return
		}
		assoc := expression.Sub.(*ast.AssocNode)
		if !irLocalIs(assoc.Target, u.target) ||
			len(assoc.Entries) == 0 {
			u.safe = false
			return
		}
		for _, entry := range assoc.Entries {
			u.scanRead(entry.Key)
			u.scanRead(entry.Val)
		}
		u.updates++
		u.updateNodes = append(u.updateNodes, expression)
	}
}

// scanRead accepts only operations that cannot retain the map or expose its
// transient identity. Additions here must be backed by the corresponding
// transient-map interface.
func (u *ownedMapUsage) scanRead(node *ast.Node) {
	if node == nil || !u.safe {
		return
	}
	if irLocalIs(node, u.target) {
		u.safe = false
		return
	}

	switch node.Op {
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if len(call.Args) > 0 &&
			irLocalIs(call.Args[0], u.target) &&
			isOwnedMapReadHostCall(call) {
			facts := u.ir.facts[node]
			facts.OwnedMapGet = true
			u.ir.facts[node] = facts
			for _, argument := range call.Args[1:] {
				u.scanRead(argument)
			}
			return
		}
	case ast.OpKeywordLookup:
		lookup := node.Sub.(*ast.KeywordLookupNode)
		if irLocalIs(lookup.Target, u.target) {
			u.scanRead(lookup.Default)
			return
		}
	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		if len(invoke.Args) > 0 &&
			irLocalIs(invoke.Args[0], u.target) &&
			isOwnedMapReadCoreCall(invoke) {
			u.scanRead(invoke.Fn)
			for _, argument := range invoke.Args[1:] {
				u.scanRead(argument)
			}
			return
		}
	}

	for _, child := range irChildren(node) {
		u.scanRead(child)
	}
}

func isOwnedMapReadHostCall(call *ast.HostCallNode) bool {
	if call == nil || call.Method == nil ||
		call.Target == nil || call.Target.Op != ast.OpConst {
		return false
	}
	target := call.Target.Sub.(*ast.ConstNode).HostSymbol
	return target != nil &&
		target.String() == "github.com:glojurelang:glojure:pkg:runtime.RT" &&
		strings.EqualFold(call.Method.Name(), "Get")
}

func isOwnedMapReadCoreCall(invoke *ast.InvokeNode) bool {
	_, name, core := irCoreCall(invoke)
	if !core {
		return false
	}
	switch name {
	case "get", "contains?", "count":
		return true
	default:
		return false
	}
}
