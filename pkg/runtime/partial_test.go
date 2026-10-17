package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestPartialOneAritiesAndMetadata(t *testing.T) {
	apply := lang.NewVar(lang.FindOrCreateNamespace(lang.NewSymbol("partial-test")),
		lang.NewSymbol("apply"))
	apply.BindRoot(nativeCoreApply{})
	fn := lang.FnFunc(func(args ...any) any { return lang.NewVector(args...) })
	p := &partialOne{fn: fn, first: "bound", apply: apply}
	for count := 0; count <= 9; count++ {
		args := make([]any, count)
		for i := range args {
			args[i] = i
		}
		want := lang.NewVector(append([]any{"bound"}, args...)...)
		for _, result := range []any{p.Invoke(args...), p.ApplyTo(lang.Seq(args))} {
			if !lang.Equals(result, want) {
				t.Fatalf("partial with %d arguments = %v, want %v", count, result, want)
			}
		}
	}
	meta := lang.NewMap(lang.NewKeyword("tag"), true)
	wrapped := p.WithMeta(meta).(lang.IFn)
	if !lang.IsFn(p) || wrapped.(lang.IMeta).Meta() != meta || p.Meta() != nil {
		t.Fatal("partial lost function identity or metadata semantics")
	}
	if got := lang.Apply1(wrapped, 42); !lang.Equals(got, lang.NewVector("bound", 42)) {
		t.Fatalf("metadata-wrapped partial = %v", got)
	}
	apply.BindRoot(lang.FnFunc(func(args ...any) any { return "redefined" }))
	if got := p.Invoke(1, 2, 3, 4); got != "redefined" {
		t.Fatalf("partial ignored apply redefinition: %v", got)
	}
}
