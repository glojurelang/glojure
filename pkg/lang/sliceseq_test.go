package lang

import (
	"sync"
	"testing"
)

func TestSliceSeqNativeAndHostBackingViews(t *testing.T) {
	values := []any{int64(1), int64(2), int64(3)}
	seq := NewSliceSeq(values)
	tail := seq.Next()
	values[1] = int64(42)
	if tail.First() != int64(42) || seq.Next().First() != int64(42) {
		t.Fatal("native slice suffix lost its backing view")
	}
	for _, input := range []any{
		[]any{int64(1), int64(2), int64(3)},
		[]int64{1, 2, 3}, [3]int64{1, 2, 3},
	} {
		seq := NewSliceSeq(input).(*SliceSeq)
		add := FnFunc2(func(a, b any) any { return a.(int64) + b.(int64) })
		if got := seq.Reduce(add); got != int64(6) {
			t.Fatalf("Reduce(%T) = %v, want 6", input, got)
		}
		if got := seq.ReduceInit(add, int64(10)); got != int64(16) {
			t.Fatalf("ReduceInit(%T) = %v, want 16", input, got)
		}
	}
}

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

func TestSliceSeqShortTailDoesNotAllocateRepeatedly(t *testing.T) {
	values := []int64{1, 2, 3}
	seq := NewSliceSeq(values)
	tail := seq.Next()
	if allocations := testing.AllocsPerRun(1000, func() { seq.Next() }); allocations != 0 {
		t.Fatalf("repeated Next allocated %v objects", allocations)
	}
	values[1] = 42
	if tail.First() != int64(42) || seq.Next().First() != int64(42) {
		t.Fatal("cached suffix lost its backing array view")
	}
	large := NewSliceSeq(make([]int, 1000)).(*SliceSeq)
	large.Next()
	if large.next.Load() != nil {
		t.Fatal("large array retained an unbounded suffix chain")
	}
}

func TestSliceSeqConcurrentTailsAndHashes(t *testing.T) {
	seq := NewSliceSeq([]any{int64(1), int64(2), int64(3)})
	want := NewList(int64(2), int64(3))
	wantHash, wantHashEq := Hash(want), HashEq(want)
	meta := NewMap(NewKeyword("test"), true)
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			tail := seq.Next().(*SliceSeq)
			if tail.First() != int64(2) || tail.Count() != 2 {
				t.Error("incorrect cached suffix")
			}
			if tail.Hash() != wantHash || tail.HashEq() != wantHashEq {
				t.Error("incorrect suffix hash")
			}
			tagged := tail.WithMeta(meta).(*SliceSeq)
			if tagged.Meta() != meta || tagged.Hash() != wantHash || tagged.HashEq() != wantHashEq {
				t.Error("metadata copy lost suffix content or hashes")
			}
		}()
	}
	close(start)
	workers.Wait()
}
