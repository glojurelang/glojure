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

func TestLoadedNamespacesUseFreshRuntimeState(t *testing.T) {
	core := lang.FindOrCreateNamespace(lang.NewSymbol("clojure.core"))
	loadedLibs := core.Intern(lang.NewSymbol("*loaded-libs*"))

	initializer, ok := runtimeStateInitializer(loadedLibs)
	if !ok {
		t.Fatal("*loaded-libs* does not have a runtime-state initializer")
	}
	if want := "lang.NewRef(lang.NewSet())"; initializer != want {
		t.Fatalf("*loaded-libs* initializer = %q, want %q", initializer, want)
	}
}

func TestRuntimeFunctionMeta(t *testing.T) {
	foo := lang.NewKeyword("foo")
	bar := lang.NewKeyword("bar")
	retTag := lang.NewKeyword("rettag")

	if got := runtimeFunctionMeta(nil); got != nil {
		t.Fatalf("nil metadata became %v", got)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil)); got != nil {
		t.Fatalf("compiler-only metadata became %v", got)
	}

	explicit := lang.NewMap(foo, bar)
	if got := runtimeFunctionMeta(explicit); !lang.Equals(got, explicit) {
		t.Fatalf("explicit metadata became %v, want %v", got, explicit)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil, foo, bar)); !lang.Equals(got, explicit) {
		t.Fatalf("mixed metadata became %v, want %v", got, explicit)
	}
}
