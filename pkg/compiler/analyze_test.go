package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestResolvedHostConstantsRetainTheirSymbols(t *testing.T) {
	nsSym := lang.NewSymbol("compiler.resolved-host-test")
	ns := lang.FindOrCreateNamespace(nsSym)
	t.Cleanup(func() { lang.RemoveNamespace(nsSym) })

	fnSym := lang.NewSymbol("example.com:host.Adapter")
	constantSym := lang.NewSymbol("example.com:host.Constant")
	analyzer := &Analyzer{
		Macroexpand1:  func(form interface{}) (interface{}, error) { return form, nil },
		FindNamespace: func(*lang.Symbol) *lang.Namespace { return ns },
		ResolveHost: func(sym *lang.Symbol) (interface{}, bool) {
			switch sym.String() {
			case fnSym.String():
				return lang.FnFunc(func(...any) any { return nil }), true
			case constantSym.String():
				return int64(42), true
			default:
				return nil, false
			}
		},
	}
	env := lang.NewMap(lang.KWNS, nsSym).(Env)

	fnNode, err := analyzer.analyzeSymbol(fnSym, env)
	if err != nil {
		t.Fatal(err)
	}
	if fnNode.Op != ast.OpConst {
		t.Fatalf("resolved host function op = %v, want OpConst", fnNode.Op)
	}
	if got := fnNode.Sub.(*ast.ConstNode).HostSymbol; !lang.Equals(got, fnSym) {
		t.Fatalf("resolved host function symbol = %v, want %v", got, fnSym)
	}

	constantNode, err := analyzer.analyzeSymbol(constantSym, env)
	if err != nil {
		t.Fatal(err)
	}
	if constantNode.Op != ast.OpConst {
		t.Fatalf("resolved host constant op = %v, want OpConst", constantNode.Op)
	}
	if got := constantNode.Sub.(*ast.ConstNode).HostSymbol; !lang.Equals(got, constantSym) {
		t.Fatalf("resolved host constant symbol = %v, want %v", got, constantSym)
	}
}

func TestInlineExpansionSupportedChecksHostMethodArity(t *testing.T) {
	resolveNumbers := func(sym *lang.Symbol) (interface{}, bool) {
		if sym.String() == "test/Numbers" {
			return lang.Numbers, true
		}
		return nil, false
	}
	dot := lang.NewSymbol(".")
	target := lang.NewSymbol("test/Numbers")
	minus := lang.NewSymbol("minus")

	unary := lang.NewList(dot, target, lang.NewList(minus, int64(1)))
	if inlineExpansionSupported(unary, resolveNumbers) {
		t.Fatal("binary Numbers.minus must not be accepted as a unary inline call")
	}

	binary := lang.NewList(dot, target, lang.NewList(minus, int64(2), int64(1)))
	if !inlineExpansionSupported(binary, resolveNumbers) {
		t.Fatal("binary Numbers.minus should be accepted")
	}

	unresolved := lang.NewList(dot, lang.NewSymbol("missing/Host"), lang.NewList(minus, int64(2), int64(1)))
	if inlineExpansionSupported(unresolved, resolveNumbers) {
		t.Fatal("an unresolved host target must retain the original Var call")
	}
}

func TestFoldLiteralNumberCall(t *testing.T) {
	method, ok := lang.FieldOrMethod(lang.Numbers, "Add")
	if !ok {
		t.Fatal("Numbers.Add was not resolved")
	}
	call := &ast.HostCallNode{
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
	}
	if got, folded := foldLiteralNumberCall(call); !folded || got != int64(5) {
		t.Fatalf("foldLiteralNumberCall = (%v, %v), want (5, true)", got, folded)
	}

	call.Args[1] = &ast.Node{
		Op:  ast.OpLocal,
		Sub: &ast.LocalNode{Name: lang.NewSymbol("x")},
	}
	if _, folded := foldLiteralNumberCall(call); folded {
		t.Fatal("call with a nonliteral argument was folded")
	}
}

func TestFoldLiteralNumberCallLeavesTrapAtRuntime(t *testing.T) {
	method, ok := lang.FieldOrMethod(lang.Numbers, "Quotient")
	if !ok {
		t.Fatal("Numbers.Quotient was not resolved")
	}
	call := &ast.HostCallNode{
		Target: &ast.Node{
			Op:  ast.OpConst,
			Sub: &ast.ConstNode{Value: lang.Numbers},
		},
		Method: lang.NewSymbol("Quotient"),
		Args: []*ast.Node{
			{Op: ast.OpConst, Sub: &ast.ConstNode{Value: int64(1)}},
			{Op: ast.OpConst, Sub: &ast.ConstNode{Value: int64(0)}},
		},
		ResolvedMethod: method,
	}
	if _, folded := foldLiteralNumberCall(call); folded {
		t.Fatal("trapping constant expression was folded")
	}
}

func TestFoldLiteralIf(t *testing.T) {
	analyzer := &Analyzer{}
	tests := []struct {
		name string
		test interface{}
		want int64
	}{
		{name: "truthy", test: int64(0), want: 1},
		{name: "false", test: false, want: 2},
		{name: "nil", test: nil, want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := lang.NewList(
				lang.NewSymbol("if"),
				test.test,
				int64(1),
				int64(2),
			)
			node, err := analyzer.parseIf(form, nil)
			if err != nil {
				t.Fatal(err)
			}
			if node.Op != ast.OpConst {
				t.Fatalf("folded if op = %v, want OpConst", node.Op)
			}
			if got := node.Sub.(*ast.ConstNode).Value; got != test.want {
				t.Fatalf("folded if = %v, want %v", got, test.want)
			}
		})
	}
}

func TestContainsResidualUnquote(t *testing.T) {
	form := lang.NewList(
		lang.NewSymbol("."),
		lang.NewSymbol("test/Host"),
		lang.NewList(
			lang.NewSymbol("method"),
			lang.NewList(lang.NewSymbol("clojure.core/unquote"), lang.NewSymbol("x")),
		),
	)
	if !containsResidualUnquote(form) {
		t.Fatal("expected residual unquote to be detected")
	}
}

func TestRuntimeFormMeta(t *testing.T) {
	userKey := lang.NewKeyword("user")
	userValue := lang.NewKeyword("value")

	if got := runtimeFormMeta(nil); got != nil {
		t.Fatalf("nil metadata became %v", got)
	}

	sourceMeta := lang.NewMap(
		lang.KWFile, "source.glj",
		lang.KWLine, 10,
		lang.KWColumn, 20,
		lang.KWEndLine, 11,
		lang.KWEndColumn, 30,
	)
	if got := runtimeFormMeta(sourceMeta); got != nil {
		t.Fatalf("source metadata became runtime metadata: %v", got)
	}

	mixed := lang.Assoc(sourceMeta, userKey, userValue).(lang.IPersistentMap)
	want := lang.NewMap(userKey, userValue)
	if got := runtimeFormMeta(mixed); !lang.Equals(got, want) {
		t.Fatalf("mixed metadata became %v, want %v", got, want)
	}
}
