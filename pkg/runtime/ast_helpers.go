package runtime

import "github.com/glojurelang/glojure/pkg/ast"

func nodeRecurs(n *ast.Node, loopID string) bool {
	switch n.Op {
	case ast.OpRecur:
		recurNode := n.Sub.(*ast.RecurNode)
		return recurNode.LoopID.Name() == loopID
	case ast.OpDo:
		doNode := n.Sub.(*ast.DoNode)
		return nodeRecurs(doNode.Ret, loopID)
	case ast.OpLet, ast.OpLoop:
		letNode := n.Sub.(*ast.LetNode)
		return nodeRecurs(letNode.Body, loopID)
	case ast.OpLetFn:
		letFnNode := n.Sub.(*ast.LetFnNode)
		return nodeRecurs(letFnNode.Body, loopID)
	case ast.OpIf:
		ifNode := n.Sub.(*ast.IfNode)
		return nodeRecurs(ifNode.Then, loopID) || nodeRecurs(ifNode.Else, loopID)
	case ast.OpTry:
		tryNode := n.Sub.(*ast.TryNode)
		if nodeRecurs(tryNode.Body, loopID) {
			return true
		}
		for _, catch := range tryNode.Catches {
			if nodeRecurs(catch, loopID) {
				return true
			}
		}
	case ast.OpCatch:
		catchNode := n.Sub.(*ast.CatchNode)
		return nodeRecurs(catchNode.Body, loopID)
	case ast.OpCase:
		caseNode := n.Sub.(*ast.CaseNode)
		if nodeRecurs(caseNode.Default, loopID) {
			return true
		}
		for _, entry := range caseNode.Entries {
			if nodeRecurs(entry.ResultExpr, loopID) {
				return true
			}
		}
	default:
		return false
	}

	return false
}
