package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

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

func TestRuntimeFunctionFormMeta(t *testing.T) {
	userKey := lang.NewKeyword("user")
	userValue := lang.NewKeyword("value")

	if got := runtimeFunctionFormMeta(nil); got != nil {
		t.Fatalf("nil metadata became %v", got)
	}

	sourceMeta := lang.NewMap(
		lang.KWFile, "source.glj",
		lang.KWLine, 10,
		lang.KWColumn, 20,
		lang.KWEndLine, 11,
		lang.KWEndColumn, 30,
	)
	if got := runtimeFunctionFormMeta(sourceMeta); got != nil {
		t.Fatalf("source metadata became runtime metadata: %v", got)
	}

	mixed := lang.Assoc(sourceMeta, userKey, userValue).(lang.IPersistentMap)
	want := lang.NewMap(userKey, userValue)
	if got := runtimeFunctionFormMeta(mixed); !lang.Equals(got, want) {
		t.Fatalf("mixed metadata became %v, want %v", got, want)
	}
}
