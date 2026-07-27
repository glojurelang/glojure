package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type testOptimizationPass struct {
	name    string
	rewrite func(*ast.Node) (*ast.Node, error)
}

func (p testOptimizationPass) Name() string {
	return p.name
}

func (p testOptimizationPass) Optimize(node *ast.Node) (*ast.Node, error) {
	return p.rewrite(node)
}

func TestOptimizerAppliesPassesInOrder(t *testing.T) {
	root := &ast.Node{
		Op:  ast.OpConst,
		Sub: &ast.ConstNode{Value: "start"},
	}
	appendValue := func(suffix string) testOptimizationPass {
		return testOptimizationPass{
			name: suffix,
			rewrite: func(root *ast.Node) (*ast.Node, error) {
				return ast.Transform(root, func(node *ast.Node) (*ast.Node, error) {
					if node.Op == ast.OpConst {
						value := node.Sub.(*ast.ConstNode).Value.(string)
						node.Sub.(*ast.ConstNode).Value = value + suffix
					}
					return node, nil
				})
			},
		}
	}
	result, err := NewOptimizer(
		appendValue("-first"),
		appendValue("-second"),
	).Optimize(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Sub.(*ast.ConstNode).Value; got != "start-first-second" {
		t.Fatalf("optimized value = %q, want ordered pass output", got)
	}
}

func TestEmptyOptimizerLeavesTreeUntouched(t *testing.T) {
	root := &ast.Node{Op: ast.OpConst, Sub: &ast.ConstNode{Value: int64(1)}}
	result, err := NewOptimizer().Optimize(root)
	if err != nil {
		t.Fatal(err)
	}
	if result != root {
		t.Fatal("empty optimizer replaced the AST root")
	}
}

func TestDefaultOptimizerFoldsNestedLiteralNumberCall(t *testing.T) {
	method, ok := lang.FieldOrMethod(lang.Numbers, "Add")
	if !ok {
		t.Fatal("Numbers.Add was not resolved")
	}
	form := lang.NewList(lang.NewSymbol("+"), int64(2), int64(3))
	raw := lang.NewList(lang.NewSymbol("original"))
	call := &ast.Node{
		Op:       ast.OpHostCall,
		Form:     form,
		RawForms: []interface{}{raw},
		Env:      lang.NewMap(lang.KWLine, int64(10)),
		Sub: &ast.HostCallNode{
			Target: &ast.Node{
				Op:  ast.OpConst,
				Sub: &ast.ConstNode{Value: lang.Numbers},
			},
			Method: lang.NewSymbol("Add"),
			Args: []*ast.Node{
				{Op: ast.OpConst, Sub: &ast.ConstNode{Value: int64(2)}},
				{Op: ast.OpConst, Sub: &ast.ConstNode{Value: int64(3)}},
			},
			ResolvedMethod: method,
		},
	}
	root := &ast.Node{
		Op:  ast.OpVector,
		Sub: &ast.VectorNode{Items: []*ast.Node{call}},
	}

	result, err := defaultOptimizer().Optimize(root)
	if err != nil {
		t.Fatal(err)
	}
	folded := result.Sub.(*ast.VectorNode).Items[0]
	if folded.Op != ast.OpConst {
		t.Fatalf("optimized op = %v, want OpConst", folded.Op)
	}
	if value := folded.Sub.(*ast.ConstNode).Value; value != int64(5) {
		t.Fatalf("optimized value = %v, want 5", value)
	}
	if folded.Form != form || folded.Env != call.Env ||
		len(folded.RawForms) != 1 || folded.RawForms[0] != raw {
		t.Fatal("constant fold did not retain AST source metadata")
	}
}
