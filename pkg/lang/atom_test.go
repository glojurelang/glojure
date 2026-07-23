package lang

import "testing"

func TestAtomCompareAndSetUsesIdentity(t *testing.T) {
	value := []int{1, 2, 3}
	atom := NewAtom(value)

	if atom.CompareAndSet([]int{1, 2, 3}, "wrong") {
		t.Fatal("compare-and-set accepted an equal but non-identical slice")
	}
	if !atom.CompareAndSet(value, "updated") {
		t.Fatal("compare-and-set rejected the identical slice")
	}
	if got := atom.Deref(); got != "updated" {
		t.Fatalf("atom value = %v, want updated", got)
	}
}

func TestAtomFixedAritySwapAllocatesOnlyNewState(t *testing.T) {
	atom := NewAtom(int64(0))
	increment := FnFunc1(func(value any) any {
		return value.(int64) + 1
	})

	if got := testing.AllocsPerRun(1_000, func() {
		atom.Swap0(increment)
	}); got > 1 {
		t.Fatalf("Swap0 allocated %v objects, want at most 1", got)
	}
}
