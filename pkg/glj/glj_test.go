package glj

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestGLJ(t *testing.T) {
	mp := Var("clojure.core", "map")
	inc := Var("clojure.core", "inc")
	res := lang.PrintString(mp.Invoke(inc, Read("[1 2 3]")))
	if res != "(2 3 4)" {
		t.Errorf("Expected (2 3 4), got %v", res)
	}
}

func TestCoreSlurpLazilyLoadsIONamespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte("lazy io"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := Var("clojure.core", "slurp").Invoke(path); got != "lazy io" {
		t.Fatalf("slurp returned %v, want %q", got, "lazy io")
	}
}

// TestFnQmarkRecognizesFnFuncN verifies that clojure.core/fn? returns true
// for all FnFuncN fixed-arity types introduced in Round 4 optimizations.
// Previously fn? only checked for *runtime.Fn and FnFunc (variadic), causing
// "Unknown type" panics in yamlstar's parser when combinators compiled to
// FnFuncN were passed to typeof*.
func TestFnQmarkRecognizesFnFuncN(t *testing.T) {
	fnQ := Var("clojure.core", "fn?")

	cases := []struct {
		name string
		val  any
		want bool
	}{
		{"FnFunc", lang.FnFunc(func(args ...any) any { return nil }), true},
		{"FnFunc0", lang.FnFunc0(func() any { return nil }), true},
		{"FnFunc1", lang.FnFunc1(func(a any) any { return a }), true},
		{"FnFunc2", lang.FnFunc2(func(a, b any) any { return a }), true},
		{"FnFunc3", lang.FnFunc3(func(a, b, c any) any { return a }), true},
		{"FnFunc4", lang.FnFunc4(func(a, b, c, d any) any { return a }), true},
		{"string", "hello", false},
		{"int", 42, false},
		{"nil", nil, false},
		{"map", lang.NewMap(), false},
	}

	for _, tc := range cases {
		got := lang.IsTruthy(fnQ.Invoke(tc.val))
		if got != tc.want {
			t.Errorf("fn?(%s): expected %v, got %v", tc.name, tc.want, got)
		}
	}
}
