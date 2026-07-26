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

func TestAtomIdenticalUpdatesAvoidNewStateAllocation(t *testing.T) {
	value := &struct{}{}
	atom := NewAtom(value)

	if got := testing.AllocsPerRun(1_000, func() {
		atom.Reset(value)
	}); got != 0 {
		t.Fatalf("identical Reset allocated %v objects, want 0", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		if !atom.CompareAndSet(value, value) {
			t.Fatal("identical CompareAndSet failed")
		}
	}); got != 0 {
		t.Fatalf("identical CompareAndSet allocated %v objects, want 0", got)
	}
}

func TestAtomIdenticalUpdatesStillNotifyWatches(t *testing.T) {
	value := &struct{}{}
	atom := NewAtom(value)
	calls := 0
	atom.AddWatch("watch", FnFunc4(func(_, _, oldValue, newValue any) any {
		calls++
		if oldValue != value || newValue != value {
			t.Fatal("watch received the wrong value")
		}
		return nil
	}))

	atom.Reset(value)
	if !atom.CompareAndSet(value, value) {
		t.Fatal("identical CompareAndSet failed")
	}
	if calls != 2 {
		t.Fatalf("watch called %d times, want 2", calls)
	}
}
