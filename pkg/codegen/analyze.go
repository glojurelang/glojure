package codegen

import (
	"github.com/glojurelang/glojure/pkg/ast"
)

// return true to stop visiting
type visitor func(*ast.Node) bool

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
	case ast.OpIf:
		ifNode := n.Sub.(*ast.IfNode)
		return nodeRecurs(ifNode.Then, loopID) || nodeRecurs(ifNode.Else, loopID)
		// TODO: review all other node types
	default:
		return false // can't recur in this node type
	}
}
