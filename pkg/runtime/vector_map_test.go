package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

type vectorMapSource struct{ seq lang.ISeq }

func (s *vectorMapSource) Seq() lang.ISeq { return s.seq }

func (*vectorMapSource) ReduceInit(lang.IFn, any) any {
	panic("map must use Seq rather than the source reduction protocol")
}

func TestVectorMapUsesSequenceOrderAndCallbackCalls(t *testing.T) {
	for _, size := range []int{0, 1, 2, 31, 32, 33, 100} {
		values := make([]any, size)
		for i := range values {
			values[i] = i
		}
		sources := []any{lang.NewVector(values...), lang.NewList(values...),
			&vectorMapSource{lang.NewSliceSeq(values)}}
		for _, source := range sources {
			calls := 0
			mapper := lang.FnFunc1(func(value any) any {
				if value != calls {
					t.Fatal("mapper changed sequence order")
				}
				calls++
				return value
			})
			result := VectorMap(mapper, source).(lang.IPersistentVector)
			if calls != size || !lang.Equals(result, lang.NewVector(values...)) {
				t.Fatalf("mapped vector at %d: calls=%d, result=%v", size, calls, result)
			}
		}
	}
}

func TestVectorMapStopsOnCallbackPanic(t *testing.T) {
	calls := 0
	defer func() {
		if recover() != "stop" || calls != 3 {
			t.Fatal("callback panic changed traversal")
		}
	}()
	VectorMap(lang.FnFunc1(func(value any) any {
		calls++
		if value == int64(2) {
			panic("stop")
		}
		return value
	}), lang.NewVector(int64(0), int64(1), int64(2), int64(3)))
}

func TestVectorMapcatSeqMapOrderAndOtherSources(t *testing.T) {
	a, b := lang.NewKeyword("a"), lang.NewKeyword("b")
	shaped := lang.NewStaticKeywordMap(lang.NewKeywordMapShape("b", "a"), 2, nil)
	for _, source := range []any{
		lang.NewMap(), lang.NewMap(a, nil, b, 2), shaped,
		shaped.Assoc(a, 3), lang.NewPersistentHashMap(a, nil, b, 2),
		lang.NewVector(lang.NewVector(a, 1), nil, lang.NewList(b, 2)),
	} {
		seqFn := lang.FnFunc1(func(value any) any { return lang.Seq(value) })
		want := lang.CreateLazilyPersistentVector(nativeMapcatSeq(seqFn, source))
		got := VectorMapcatSeq(seqFn, source)
		if !lang.Equals(got, want) {
			t.Fatalf("flattening %T: %v, want %v", source, got, want)
		}
	}
}
