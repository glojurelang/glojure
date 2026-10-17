package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeMapOverVectorRealizesAndCachesChunks(t *testing.T) {
	values := make([]any, 100)
	for i := range values {
		values[i] = int64(i)
	}
	calls := 0
	mapper := lang.FnFunc1(func(value any) any {
		calls++
		return value.(int64) + 1
	})
	seq := nativeMapSeq(mapper, lang.NewVector(values...)).(lang.ISeq)
	if calls != 0 {
		t.Fatal("map realized its vector input eagerly")
	}
	if first := seq.First(); first != int64(1) || calls != 32 {
		t.Fatalf("first = %v with %d calls, want 1 and 32", first, calls)
	}
	for range 2 {
		if count := lang.Count(seq); count != 100 || calls != 100 {
			t.Fatalf("count = %d with %d calls, want 100 and 100", count, calls)
		}
		if result := lang.CreateLazilyPersistentVector(seq); result.Nth(99) != int64(100) || calls != 100 {
			t.Fatal("vector construction reran mapped elements")
		}
	}
}
