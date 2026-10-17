package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

func TestConcatPairLazinessChunksAndFinalTailIdentity(t *testing.T) {
	calls := 0
	tail := lang.NewLazySeq(func() any { calls++; return lang.NewList(3, 4) })
	joined := nativeConcatPair(lang.NewVector(1, 2), tail)
	if calls != 0 {
		t.Fatal("concat realized its right input during construction")
	}
	if joined.First() != 1 || calls != 0 {
		t.Fatal("concat forced its final tail early")
	}
	if !lang.Equals(lang.NewVector(lang.ToSlice(joined)...), lang.NewVector(1, 2, 3, 4)) || calls != 1 {
		t.Fatal("concat lost chunk values or repeated its final tail")
	}
	right := lang.NewList(3, 4)
	if lang.Seq(nativeConcatPair(nil, right)) != right {
		t.Fatal("concat copied its final tail")
	}
}

func TestConcatPairSkipsEmptyInputChunks(t *testing.T) {
	source := lang.NewChunkedCons(lang.NewSliceChunk(nil), lang.NewList(1, 2))
	result := nativeConcatPair(source, lang.NewList(3))
	if !lang.Equals(lang.NewVector(lang.ToSlice(result)...), lang.NewVector(1, 2, 3)) {
		t.Fatal("concat retained an empty input chunk")
	}
}
