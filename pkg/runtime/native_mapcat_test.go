package runtime

import (
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeMapcatIsLazyAndCachesValues(t *testing.T) {
	calls := 0
	inputs := 0
	source := lang.NewLazySeq(func() any {
		inputs++
		return lang.NewList(int64(1), int64(2), int64(3))
	})
	fn := lang.FnFunc1(func(value any) any {
		calls++
		n := value.(int64)
		return lang.NewVector(n, n+10)
	})
	result := nativeMapcatSeq(fn, source).(lang.ISeq)
	if calls != 0 || inputs != 0 {
		t.Fatal("mapcat realized its input eagerly")
	}
	for i := 0; i < 2; i++ {
		if got := result.First(); got != int64(1) {
			t.Fatalf("first = %v, want 1", got)
		}
	}
	if calls != 1 || inputs != 1 {
		t.Fatalf("first realization: %d mapping calls, %d input calls", calls, inputs)
	}
	want := []any{int64(1), int64(11), int64(2), int64(12), int64(3), int64(13)}
	if got := lang.ToSlice(result); !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %v, want %v", got, want)
	}
	if calls != 3 {
		t.Fatalf("mapping ran %d times, want 3", calls)
	}
}

func TestNativeMapcatKeepsChunkedMapping(t *testing.T) {
	calls := 0
	fn := lang.FnFunc1(func(n any) any {
		calls++
		return lang.NewList(n)
	})
	result := nativeMapcatSeq(fn, lang.NewLongRange(0, 100, 1)).(lang.ISeq)
	if calls != 0 {
		t.Fatal("chunked mapcat realized eagerly")
	}
	if got := result.First(); got != int64(0) || calls != 32 {
		t.Fatalf("first = %v, mapping calls = %d, want 0 and 32", got, calls)
	}
	if got := result.Next().First(); got != int64(1) || calls != 32 {
		t.Fatalf("second = %v, mapping calls = %d, want 1 and 32", got, calls)
	}
}

func TestNativeMapcatSkipsEmptyCollections(t *testing.T) {
	fn := lang.FnFunc1(func(value any) any {
		n := value.(int64)
		switch n % 3 {
		case 0:
			return lang.NewList(n)
		case 1:
			return nil
		default:
			return lang.NewVector()
		}
	})
	result := nativeMapcatSeq(fn, lang.NewLongRange(0, 10000, 1))
	got := lang.ToSlice(result)
	if len(got) != 3334 || got[0] != int64(0) || got[len(got)-1] != int64(9999) {
		t.Fatalf("unexpected flattened range: %v", got)
	}
	if got := lang.Seq(nativeMapcatSeq(fn, nil)); got != nil {
		t.Fatalf("empty mapcat = %v, want nil", got)
	}
}

func TestNativeMapcatPreservesOutputChunks(t *testing.T) {
	fn := lang.FnFunc1(func(n any) any {
		values := lang.NewChunkedCons(lang.NewSliceChunk([]any{n}), nil)
		return lang.NewChunkedCons(lang.NewSliceChunk(nil), values)
	})
	result := nativeMapcatSeq(fn, lang.NewList(int64(1), int64(2)))
	want := []any{int64(1), int64(2)}
	if got := lang.ToSlice(result); !reflect.DeepEqual(got, want) {
		t.Fatalf("chunked result = %v, want %v", got, want)
	}
}

func TestNativeMapcatFinishesOutputBeforeMappingNextChunk(t *testing.T) {
	calls := 0
	fn := lang.FnFunc1(func(n any) any {
		calls++
		if n == int64(31) {
			return lang.NewLongRange(0, 50, 1)
		}
		return nil
	})
	result := lang.Seq(nativeMapcatSeq(fn, lang.NewLongRange(0, 100, 1)))
	for i := int64(0); i < 50; i++ {
		if result == nil || result.First() != i || calls != 32 {
			t.Fatalf("element %d: result %v, mapping calls %d", i, result, calls)
		}
		result = result.Next()
	}
	if result != nil || calls != 100 {
		t.Fatalf("end: result %v, mapping calls %d", result, calls)
	}
}
