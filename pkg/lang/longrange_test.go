package lang

import "testing"

type stoppingInt64Reducer struct {
	FnFunc2
	calls     int
	stopAfter int
}

func (r *stoppingInt64Reducer) ReduceInt64Step(
	acc, value int64,
) (int64, bool) {
	r.calls++
	return acc + value, r.calls == r.stopAfter
}

func TestLongRangeChunksAreBounded(t *testing.T) {
	r := NewLongRange(0, 100, 1).(*LongRange)

	first := r.ChunkedFirst()
	if got := first.Count(); got != longRangeChunkSize {
		t.Fatalf("first chunk count = %d, want %d", got, longRangeChunkSize)
	}
	if got := first.Nth(longRangeChunkSize - 1); got != int64(31) {
		t.Fatalf("last value in first chunk = %v, want 31", got)
	}

	more := r.ChunkedMore().(*LongRange)
	if got := more.First(); got != int64(32) {
		t.Fatalf("first value after first chunk = %v, want 32", got)
	}
	if got := more.Count(); got != 68 {
		t.Fatalf("remaining count = %d, want 68", got)
	}

	tail := more.ChunkedMore().(*LongRange).ChunkedMore().(*LongRange)
	if got := tail.ChunkedFirst().Count(); got != 4 {
		t.Fatalf("tail chunk count = %d, want 4", got)
	}
	if tail.ChunkedNext() != nil {
		t.Fatal("tail ChunkedNext returned a sequence")
	}
	if tail.ChunkedMore() != emptyList {
		t.Fatal("tail ChunkedMore did not return the empty list")
	}
}

func TestDescendingLongRangeChunksAreBounded(t *testing.T) {
	r := NewLongRange(100, 0, -1).(*LongRange)
	first := r.ChunkedFirst()
	if got := first.Count(); got != longRangeChunkSize {
		t.Fatalf("first chunk count = %d, want %d", got, longRangeChunkSize)
	}
	if got := first.Nth(longRangeChunkSize - 1); got != int64(69) {
		t.Fatalf("last value in first chunk = %v, want 69", got)
	}

	more := r.ChunkedMore().(*LongRange)
	if got := more.First(); got != int64(68) {
		t.Fatalf("first value after first chunk = %v, want 68", got)
	}
}

func TestLongRangeInt64StepsCanStopDescendingReduction(t *testing.T) {
	r := NewLongRange(10, 0, -2).(*LongRange)
	reducer := &stoppingInt64Reducer{stopAfter: 3}
	got := r.ReduceInt64Steps(reducer, int64(0))
	if got != int64(24) {
		t.Fatalf("ReduceInt64Steps = %v, want 24", got)
	}
	if reducer.calls != 3 {
		t.Fatalf("reducer calls = %d, want 3", reducer.calls)
	}
}
