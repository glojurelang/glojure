package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

type coreLinkArityProbe struct{}

func (coreLinkArityProbe) Invoke(args ...any) any       { return len(args) }
func (p coreLinkArityProbe) ApplyTo(args lang.ISeq) any { return p.Invoke(lang.ToSlice(args)...) }

func TestCoreArityAdapterPreservesAllCallCounts(t *testing.T) {
	fn := CoreArityFn(coreLinkArityProbe{})
	results := []any{fn.Invoke0(), fn.Invoke1(0), fn.Invoke2(0, 1), fn.Invoke3(0, 1, 2),
		fn.Invoke4(0, 1, 2, 3), fn.Invoke5(0, 1, 2, 3, 4), fn.Invoke(0, 1, 2, 3, 4, 5, 6)}
	for index, result := range results {
		want := index
		if index == 6 {
			want = 7
		}
		if result != want {
			t.Fatalf("core arity adapter returned %v, want %d", result, want)
		}
	}
	if CoreArityFn(fn) != fn {
		t.Fatal("existing arity table was wrapped again")
	}
}

func TestCoreNativeCallRefreshRunsOncePerLoader(t *testing.T) {
	coreNativeLinks.Lock()
	original, done := coreNativeLinks.refresh, coreNativeLinks.done
	coreNativeLinks.Unlock()
	defer func() {
		coreNativeLinks.Lock()
		coreNativeLinks.refresh, coreNativeLinks.done = original, done
		coreNativeLinks.Unlock()
	}()
	calls := 0
	RegisterCoreNativeLinks(func() { calls++ })
	refreshCoreNativeLinks()
	refreshCoreNativeLinks()
	if calls != 1 {
		t.Fatal("core direct calls were refreshed after bootstrap")
	}
}
