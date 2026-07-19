package lang

import (
	"sync"
	"testing"
)

func TestNamespaceMappingsAreSnapshots(t *testing.T) {
	ns := NewNamespace(NewSymbol("test.namespace-snapshot"))
	first := NewSymbol("first")
	second := NewSymbol("second")
	firstVar := ns.Intern(first)

	snapshot := ns.Mappings()
	if got := snapshot.ValAt(first); got != firstVar {
		t.Fatalf("snapshot first mapping = %v, want %v", got, firstVar)
	}

	secondVar := ns.Intern(second)
	if got := snapshot.ValAt(second); got != nil {
		t.Fatalf("old snapshot contains later mapping: %v", got)
	}
	if got := ns.Mappings().ValAt(second); got != secondVar {
		t.Fatalf("current snapshot second mapping = %v, want %v", got, secondVar)
	}
	if got := ns.GetMapping(NewSymbol("second")); got != secondVar {
		t.Fatalf("semantic symbol lookup = %v, want %v", got, secondVar)
	}
}

func TestNamespaceConcurrentIntern(t *testing.T) {
	ns := NewNamespace(NewSymbol("test.namespace-concurrent-intern"))
	const goroutines = 32
	vars := make([]*Var, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func() {
			defer wg.Done()
			vars[i] = ns.Intern(NewSymbol("shared"))
		}()
	}
	wg.Wait()

	for i, vr := range vars[1:] {
		if vr != vars[0] {
			t.Fatalf("vars[%d] = %p, want %p", i+1, vr, vars[0])
		}
	}
}
