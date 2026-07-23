//go:build !glj_aot_runtime

package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestScalarReplaceableAtomUsage(t *testing.T) {
	name := lang.NewSymbol("state")
	binding := &ast.BindingNode{
		Name: name,
		Init: localAtomTestInvoke(
			"atom",
			localAtomTestConst(int64(0)),
		),
	}
	local := func() *ast.Node {
		node := ast.MakeNode(ast.OpLocal, name)
		node.Sub = &ast.LocalNode{Name: name}
		return node
	}
	body := ast.MakeNode(ast.OpDo, nil)
	body.Sub = &ast.DoNode{
		Statements: []*ast.Node{
			localAtomTestInvoke(
				"swap!",
				local(),
				localAtomTestConst(lang.FnFunc1(func(value any) any {
					return value.(int64) + 1
				})),
			),
			localAtomTestInvoke(
				"reset!",
				local(),
				localAtomTestConst(int64(42)),
			),
		},
		Ret: localAtomTestInvoke("deref", local()),
	}

	if got := scalarReplaceableAtomInit(binding, nil, body); got == nil {
		t.Fatal("non-escaping local atom was not scalar replaceable")
	}

	if got := scalarReplaceableAtomInit(binding, nil, local()); got != nil {
		t.Fatal("escaping local atom was scalar replaceable")
	}
}

func TestScalarReplaceableAtomRejectsClosureCapture(t *testing.T) {
	name := lang.NewSymbol("state")
	binding := &ast.BindingNode{
		Name: name,
		Init: localAtomTestInvoke(
			"atom",
			localAtomTestConst(int64(0)),
		),
	}
	local := ast.MakeNode(ast.OpLocal, name)
	local.Sub = &ast.LocalNode{Name: name}
	deref := localAtomTestInvoke("deref", local)
	method := ast.MakeNode(ast.OpFnMethod, nil)
	method.Sub = &ast.FnMethodNode{Body: deref}
	fn := ast.MakeNode(ast.OpFn, nil)
	fn.Sub = &ast.FnNode{Methods: []*ast.Node{method}}

	if got := scalarReplaceableAtomInit(binding, nil, fn); got != nil {
		t.Fatal("closure-captured local atom was scalar replaceable")
	}
}

func localAtomTestInvoke(name string, args ...*ast.Node) *ast.Node {
	vr := lang.NSCore.FindInternedVar(lang.NewSymbol(name))
	fn := ast.MakeNode(ast.OpVar, vr)
	fn.Sub = &ast.VarNode{Var: vr}
	node := ast.MakeNode(ast.OpInvoke, nil)
	node.Sub = &ast.InvokeNode{Fn: fn, Args: args}
	return node
}

func localAtomTestConst(value any) *ast.Node {
	node := ast.MakeNode(ast.OpConst, value)
	node.Sub = &ast.ConstNode{Value: value}
	return node
}
