package runtime

import (
	"sync"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

type distinctCollision struct{ value int }

func (distinctCollision) HashEq() uint32 { return 7 }
func (v distinctCollision) Equals(other any) bool {
	o, ok := other.(distinctCollision)
	return ok && v.value == o.value
}

func TestNativeDistinctMatchesPersistentSet(t *testing.T) {
	values := []any{
		nil, nil, false, false, int64(1), int(1), float64(1), int64(1),
		"a", "a", lang.NewVector(int64(1)), lang.NewList(int64(1)),
		distinctCollision{1}, distinctCollision{2}, distinctCollision{1},
	}
	seen := lang.NewSet()
	var want []any
	for _, value := range values {
		if !seen.Contains(value) {
			want = append(want, value)
			seen = seen.Cons(value).(*lang.Set)
		}
	}
	result := nativeDistinctSeq(lang.NewSliceSeq(values))
	if got := lang.ToSlice(result); !lang.Equals(lang.NewVector(got...), lang.NewVector(want...)) {
		t.Fatalf("distinct = %v, want %v", got, want)
	}
	if got := lang.Seq(nativeDistinctSeq(nil)); got != nil {
		t.Fatalf("empty distinct = %v, want nil", got)
	}
}

func TestNativeDistinctIsLazyAndCachesResults(t *testing.T) {
	calls := 0
	source := lang.NewMappedSeq(lang.FnFunc1(func(value any) any {
		calls++
		return value
	}), lang.NewList(int64(1), int64(1), int64(2)))
	result := nativeDistinctSeq(source).(lang.ISeq)
	if calls != 0 {
		t.Fatal("distinct realized eagerly")
	}
	for i := 0; i < 2; i++ {
		if got := result.First(); got != int64(1) {
			t.Fatalf("first = %v", got)
		}
	}
	if calls != 1 {
		t.Fatalf("first called mapper %d times, want 1", calls)
	}
	for i := 0; i < 2; i++ {
		if got := result.Next().First(); got != int64(2) {
			t.Fatalf("second = %v", got)
		}
	}
	if calls != 3 {
		t.Fatalf("second called mapper %d times, want 3", calls)
	}
}

func TestNativeDistinctConcurrentReaders(t *testing.T) {
	values := make([]any, 1000)
	for i := range values {
		values[i] = int64(i % 20)
	}
	result := nativeDistinctSeq(lang.NewSliceSeq(values))
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if got := lang.Count(result); got != 20 {
				t.Errorf("count = %d, want 20", got)
			}
		}()
	}
	workers.Wait()
}
