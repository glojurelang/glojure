package lang

import "testing"

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
