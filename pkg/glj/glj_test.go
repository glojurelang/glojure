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

func TestProtocolExtensionSurvivesDefinitionReload(t *testing.T) {
	eval := Var("clojure.core", "eval")
	result := eval.Invoke(Read(`
		(do
		  (defprotocol CodexReloadProtocol
		    (codex-reload-method [this value]))
		  (extend-protocol CodexReloadProtocol
		    nil
		    (codex-reload-method [_ value] (+ value 1))
		    go/string
		    (codex-reload-method [target value] (str target value)))
		  (let [before [(codex-reload-method nil 4)
		                (codex-reload-method "x" 7)]]
		    (defprotocol CodexReloadProtocol
		      (codex-reload-method [this value]))
		    [before
		     (codex-reload-method nil 40)
		     (codex-reload-method "y" 2)]))`))

	if got, want := lang.PrintString(result), `[[5 "x7"] 41 "y2"]`; got != want {
		t.Fatalf("protocol reload result = %s, want %s", got, want)
	}
}

// TestFnQmarkRecognizesFnFuncN verifies that clojure.core/fn? returns true
// for all FnFuncN fixed-arity types introduced in Round 4 optimizations.
// Previously fn? only checked for *runtime.Fn and FnFunc (variadic), causing
// "Unknown type" panics in yamlstar's parser when combinators compiled to
// FnFuncN were passed to typeof*.
func TestFnQmarkRecognizesFnFuncN(t *testing.T) {
	fnQ := Var("clojure.core", "fn?")
	plus := Var("clojure.core", "+").(*lang.Var)
	minus := Var("clojure.core", "-").(*lang.Var)
	str := Var("clojure.core", "str").(*lang.Var)

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
		{"FnFunc5", lang.FnFunc5(func(a, b, c, d, e any) any { return a }), true},
		{"FnFunc6", lang.FnFunc6(func(a, b, c, d, e, f any) any { return a }), true},
		{"FnFunc20", lang.FnFunc20(func(
			a0, a1, a2, a3, a4, a5, a6, a7, a8, a9,
			a10, a11, a12, a13, a14, a15, a16, a17, a18, a19 any,
		) any {
			return a0
		}), true},
		{"VariadicFn", lang.NewVariadicFn(0, func(_ []any, _ lang.ISeq) any { return nil }), true},
		{"native +", plus.Get(), true},
		{"native -", minus.Get(), true},
		{"native str", str.Get(), true},
		{"Var", plus, false},
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
