package lang

import "testing"

func TestSliceSeqIsCounted(t *testing.T) {
	for _, value := range []any{[]any{1, 2, 3}, []int{1, 2, 3}, [3]int{1, 2, 3}} {
		seq := NewSliceSeq(value)
		for remaining := 3; remaining > 0; remaining-- {
			if _, ok := seq.(Counted); !ok {
				t.Fatalf("%T slice sequence does not implement Counted", value)
			}
			if got := Count(seq); got != remaining {
				t.Fatalf("%T suffix count = %d, want %d", value, got, remaining)
			}
			if allocations := testing.AllocsPerRun(100, func() {
				Count(seq)
			}); allocations != 0 {
				t.Fatalf("Count(%T) allocated %v objects", value, allocations)
			}
			seq = seq.Next()
		}
		if seq != nil {
			t.Fatal("nonempty suffix after final element")
		}
	}
}

func TestCountStopsAtCountedTail(t *testing.T) {
	tail := NewSliceSeq(make([]int, 1000))
	value := NewCons(int64(1), tail)
	if got := Count(value); got != 1001 {
		t.Fatalf("count = %d, want 1001", got)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		Count(value)
	}); allocations != 0 {
		t.Fatalf("counted tail allocated %v objects", allocations)
	}
}
