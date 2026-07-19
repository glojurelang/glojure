package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestDirectHostMethod(t *testing.T) {
	target := &ast.Node{
		Op: ast.OpConst,
		Sub: &ast.ConstNode{
			Value: lang.Numbers,
		},
	}

	if got, ok := directHostMethod(target, "multiply", 2); !ok || got != "Multiply" {
		t.Fatalf("directHostMethod(multiply) = %q, %v", got, ok)
	}
	if got, ok := directHostMethod(target, "UncheckedIntDivide", 2); ok {
		t.Fatalf("typed method unexpectedly resolved directly as %q", got)
	}
	if got, ok := directHostMethod(target, "multiply", 1); ok {
		t.Fatalf("wrong arity unexpectedly resolved directly as %q", got)
	}
}
