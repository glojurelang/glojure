package compiler

import (
	"testing"

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
