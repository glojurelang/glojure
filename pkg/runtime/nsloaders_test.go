package runtime

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestUnboundVarLazilyLoadsAOTNamespace(t *testing.T) {
	const (
		nsName   = "glojure.test.lazy-loader"
		resource = "glojure/test/lazy-loader"
	)
	ns := lang.FindOrCreateNamespace(lang.NewSymbol(nsName))
	value := ns.Intern(lang.NewSymbol("value"))

	var loads atomic.Int32
	RegisterNSLoader(resource, func() {
		loads.Add(1)
		value.BindRoot(42)
	})

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if got := value.Get(); got != 42 {
				t.Errorf("value.Get() = %v, want 42", got)
			}
		}()
	}
	wg.Wait()

	if got := loads.Load(); got != 1 {
		t.Fatalf("loader ran %d times, want 1", got)
	}
	if got := value.Get(); got != 42 {
		t.Fatalf("second value.Get() = %v, want 42", got)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("loader ran again after namespace was loaded: %d", got)
	}
}

func TestExplicitAOTNamespaceLoadCanReload(t *testing.T) {
	const resource = "glojure/test/reloadable-loader"
	var loads int
	RegisterNSLoader(resource, func() {
		loads++
	})

	loader := GetNSLoader(resource)
	loader()
	loader()

	if loads != 2 {
		t.Fatalf("explicit loader ran %d times, want 2", loads)
	}
}
