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

func TestDirectLinkOptimizerFusesPopConj(t *testing.T) {
	coreVar := func(name string) *lang.Var {
		return lang.NSCore.Intern(lang.NewSymbol(name))
	}
	varNode := func(vr *lang.Var) *ast.Node {
		return &ast.Node{
			Op:  ast.OpVar,
			Sub: &ast.VarNode{Var: vr, Meta: vr.Meta()},
		}
	}
	collection := &ast.Node{
		Op:  ast.OpConst,
		Sub: &ast.ConstNode{Value: lang.NewVector(int64(1))},
	}
	value := &ast.Node{
		Op:  ast.OpConst,
		Sub: &ast.ConstNode{Value: int64(2)},
	}
	pop := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   varNode(coreVar("pop")),
			Args: []*ast.Node{collection},
		},
	}
	root := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   varNode(coreVar("conj")),
			Args: []*ast.Node{pop, value},
		},
	}

	result, err := NewDefaultOptimizer(OptimizationOptions{
		DirectLinking: true,
	}).Optimize(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Op != ast.OpReplaceLast {
		t.Fatalf("optimized op = %v, want OpReplaceLast", result.Op)
	}
	replace := result.Sub.(*ast.ReplaceLastNode)
	if replace.Collection != collection || replace.Value != value {
		t.Fatal("fused operation did not retain its operands")
	}
}

func TestReplaceLastFusionRequiresDirectLinking(t *testing.T) {
	conj := lang.NSCore.Intern(lang.NewSymbol("conj"))
	pop := lang.NSCore.Intern(lang.NewSymbol("pop"))
	varNode := func(vr *lang.Var) *ast.Node {
		return &ast.Node{Op: ast.OpVar, Sub: &ast.VarNode{Var: vr}}
	}
	root := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn: varNode(conj),
			Args: []*ast.Node{{
				Op: ast.OpInvoke,
				Sub: &ast.InvokeNode{
					Fn:   varNode(pop),
					Args: []*ast.Node{{Op: ast.OpConst, Sub: &ast.ConstNode{}}},
				},
			}, {Op: ast.OpConst, Sub: &ast.ConstNode{}}},
		},
	}

	result, err := NewDefaultOptimizer(OptimizationOptions{}).Optimize(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Op != ast.OpInvoke {
		t.Fatalf("optimized op = %v, want OpInvoke", result.Op)
	}
}
