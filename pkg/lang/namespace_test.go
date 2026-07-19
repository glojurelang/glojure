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

func TestNamespaceReferAll(t *testing.T) {
	source := NewNamespace(NewSymbol("refer-all-source"))
	first := source.Intern(NewSymbol("first"))
	second := source.Intern(NewSymbol("second"))
	target := NewNamespace(NewSymbol("refer-all-target"))

	target.ReferAll(source, []NamespaceReference{
		{Alias: NewSymbol("renamed"), Source: NewSymbol("first")},
		{Alias: NewSymbol("second"), Source: NewSymbol("second")},
		{Alias: NewSymbol("missing"), Source: NewSymbol("missing")},
	})

	if got := target.GetMapping(NewSymbol("renamed")); got != first {
		t.Fatalf("renamed mapping = %v, want %v", got, first)
	}
	if got := target.GetMapping(NewSymbol("second")); got != second {
		t.Fatalf("second mapping = %v, want %v", got, second)
	}
	if got := target.GetMapping(NewSymbol("missing")); got != nil {
		t.Fatalf("missing mapping = %v, want nil", got)
	}
}

func TestNamespaceReferAllSnapshotIsLazyAndStable(t *testing.T) {
	source := NewNamespace(NewSymbol("refer-snapshot-source"))
	first := source.Intern(NewSymbol("first"))
	source.Intern(NewSymbol("excluded"))
	target := NewNamespace(NewSymbol("refer-snapshot-target"))

	target.ReferAllSnapshot(source, []string{"excluded"})
	if got := target.GetMapping(NewSymbol("first")); got != first {
		t.Fatalf("snapshot first mapping = %v, want %v", got, first)
	}
	if got := target.GetMapping(NewSymbol("excluded")); got != nil {
		t.Fatalf("snapshot excluded mapping = %v, want nil", got)
	}
	if got := target.Mappings().ValAt(NewSymbol("first")); got != first {
		t.Fatalf("materialized first mapping = %v, want %v", got, first)
	}

	late := source.Intern(NewSymbol("late"))
	if got := target.GetMapping(NewSymbol("late")); got != nil {
		t.Fatalf("captured snapshot changed after source mutation: %v", got)
	}

	target.ReferAllSnapshot(source, []string{"excluded"})
	if got := target.GetMapping(NewSymbol("late")); got != late {
		t.Fatalf("refreshed snapshot late mapping = %v, want %v", got, late)
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
