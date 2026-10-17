package lang

import "testing"

func TestVectorSeqChunksAndElementTails(t *testing.T) {
	for _, count := range []int{0, 1, 31, 32, 33, 63, 64, 65, 1024, 1025} {
		items := make([]any, count)
		for i := range items {
			items[i] = i
		}
		vector := NewVector(items...)
		index := 0
		for seq := vector.Seq(); seq != nil; seq = seq.Next() {
			chunked, ok := seq.(IChunkedSeq)
			if !ok || seq.Seq() != seq {
				t.Fatalf("vector seq at %d lost chunk support", index)
			}
			chunk := chunked.ChunkedFirst()
			wantSize := min(32-index%32, count-index)
			if chunk.Count() != wantSize || seq.First() != index {
				t.Fatalf("chunk at %d/%d has size %d", index, count, chunk.Count())
			}
			for i := 0; i < chunk.Count(); i++ {
				if chunk.Nth(i) != index+i {
					t.Fatal("chunk value changed")
				}
			}
			if next := chunked.ChunkedNext(); index+wantSize == count {
				if next != nil || Seq(chunked.ChunkedMore()) != nil {
					t.Fatal("last chunk has a nonempty tail")
				}
			} else if next.First() != index+wantSize {
				t.Fatal("chunk tail starts at the wrong element")
			}
			index++
		}
		if index != count {
			t.Fatalf("vector traversal returned %d of %d items", index, count)
		}
	}
}

func TestVectorSeqMetadataAndPersistentSnapshot(t *testing.T) {
	v := NewVector(1, 2, 3)
	s := v.Seq()
	meta := NewMap(NewKeyword("tag"), true)
	withMeta := s.(IObj).WithMeta(meta).(ISeq)
	if withMeta.(IMeta).Meta() != meta || s.(IMeta).Meta() != nil {
		t.Fatal("metadata changed the original sequence")
	}
	chunk := withMeta.(IChunkedSeq).ChunkedFirst()
	v = v.AssocN(0, -1).(*Vector)
	if chunk.Nth(0) != 1 || v.Nth(0) != -1 {
		t.Fatal("chunk did not preserve its persistent vector snapshot")
	}
	if chunk.NthDefault(-1, "missing") != "missing" ||
		chunk.NthDefault(3, "missing") != "missing" {
		t.Fatal("chunk lookup ignored its bounds")
	}
}

func TestChunkedReductionStopsBeforeRealizingTail(t *testing.T) {
	tailCalls := 0
	seq := NewChunkedCons(NewSliceChunk([]any{1, 2, 3}), NewLazySeq(func() any {
		tailCalls++
		return NewList(4, 5)
	}))
	calls := 0
	reducer := FnFunc2(func(sum, value any) any {
		calls++
		if value == 2 {
			return NewReduced(sum.(int) + 2)
		}
		return sum.(int) + value.(int)
	})
	if got := seq.ReduceInit(reducer, 0); got != 3 || calls != 2 || tailCalls != 0 {
		t.Fatalf("reduce-init = %v with %d calls and %d tails", got, calls, tailCalls)
	}
	calls = 0
	if got := seq.Reduce(reducer); got != 3 || calls != 1 || tailCalls != 0 {
		t.Fatalf("reduce = %v with %d calls and %d tails", got, calls, tailCalls)
	}
	if got := NewSliceChunk(nil).ReduceInit(reducer, 42); got != 42 {
		t.Fatalf("empty chunk reduction = %v", got)
	}
}

func TestChunkedCountAndReductionWithUnchunkedTail(t *testing.T) {
	seq := NewChunkedCons(NewSliceChunk([]any{1, 2}), NewList(3, 4))
	if Count(seq) != 4 || seq.Count() != 4 {
		t.Fatal("chunked count lost unchunked tail elements")
	}
	add := FnFunc2(func(a, b any) any { return a.(int) + b.(int) })
	if seq.ReduceInit(add, 0) != 10 || seq.Reduce(add) != 10 {
		t.Fatal("chunked reduction lost unchunked tail elements")
	}
}
