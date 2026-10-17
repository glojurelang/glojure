package lang

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMappedSeqIsLazyAndCachesValues(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any {
		calls++
		return value.(int) * 2
	}), NewList(1, 2, 3))

	if calls != 0 {
		t.Fatalf("mapping ran during construction: %d calls", calls)
	}
	if mapped.(*mappedSeq).IsRealized() {
		t.Fatal("mapped sequence reported realized before access")
	}
	if got := mapped.First(); got != 2 {
		t.Fatalf("first mapped value = %v, want 2", got)
	}
	if got := mapped.First(); got != 2 {
		t.Fatalf("cached first mapped value = %v, want 2", got)
	}
	if calls != 1 {
		t.Fatalf("first mapped value ran %d times, want 1", calls)
	}

	rest := mapped.More()
	if calls != 1 {
		t.Fatalf("rest realized the next mapper: %d calls", calls)
	}
	next := mapped.Next()
	if calls != 2 || next != rest {
		t.Fatalf("next did not realize the cached tail: %d calls", calls)
	}
	if got := next.First(); got != 4 {
		t.Fatalf("second mapped value = %v, want 4", got)
	}
	if calls != 2 {
		t.Fatalf("second mapped value ran %d total times, want 2", calls)
	}
}

func TestMappedSeqDefersInputAndKeepsChunkSemantics(t *testing.T) {
	inputs, calls := 0, 0
	values := make([]any, 65)
	for i := range values {
		values[i] = i
	}
	source := NewLazySeq(func() any {
		inputs++
		return NewVector(values...)
	})
	mapped := NewLazyMappedSeq(FnFunc1(func(value any) any {
		calls++
		return value.(int) + 1
	}), source)
	if inputs != 0 || calls != 0 || mapped.(IPending).IsRealized() {
		t.Fatal("deferred map realized its input during construction")
	}
	if first := mapped.First(); first != 1 || inputs != 1 || calls != 32 {
		t.Fatalf("first = %v after %d input and %d mapper calls", first, inputs, calls)
	}
	if _, chunked := Seq(mapped).(IChunkedSeq); !chunked {
		t.Fatal("mapped vector lost chunk support")
	}
	if mapped.Next().First() != 2 || calls != 32 {
		t.Fatal("next remapped a cached chunk")
	}
	for range 2 {
		if Count(mapped) != 65 || calls != 65 || inputs != 1 {
			t.Fatalf("traversal ran %d input and %d mapper calls", inputs, calls)
		}
	}
}

func TestDeferredMappedSeqEmptyAndComposedInputs(t *testing.T) {
	for _, empty := range []any{nil, emptyList, NewVector(), NewMap()} {
		seq := NewLazyMappedSeq(FnFunc1(func(value any) any {
			t.Fatal("empty mapping invoked its mapper")
			return nil
		}), empty)
		if seq.(IPending).IsRealized() || Seq(seq) != nil || !seq.(IPending).IsRealized() {
			t.Fatal("empty mapping has incorrect realization state")
		}
	}
	calls := 0
	mapper := FnFunc1(func(value any) any {
		calls++
		return value.(int) + 1
	})
	inner := NewLazyMappedSeq(mapper, NewList(1, 2))
	outer := NewLazyMappedSeq(mapper, inner)
	if calls != 0 || outer.First() != 3 || calls != 2 {
		t.Fatalf("composed mapping ran %d mapper calls", calls)
	}
	if Count(outer) != 2 || calls != 4 || outer.First() != 3 || calls != 4 {
		t.Fatal("composed mapping did not share its cached values")
	}
}

func TestDeferredMappedSeqRetriesFailedChunk(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		calls, failed := 0, false
		mapped := NewLazyMappedSeq(FnFunc1(func(value any) any {
			calls++
			if value == failAt && !failed {
				failed = true
				panic("retry")
			}
			return value
		}), NewVector(1, 2, 3))
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing mapper panic")
				}
			}()
			mapped.First()
		}()
		if mapped.(IPending).IsRealized() {
			t.Fatal("failed chunk was published as realized")
		}
		if got := ToSlice(mapped); !reflect.DeepEqual(got, []any{1, 2, 3}) || calls != failAt+3 {
			t.Fatalf("retry at %d: result %v after %d calls", failAt, got, calls)
		}
	}
}

func TestDeferredMappedSeqConcurrentChunkReaders(t *testing.T) {
	values := make([]any, 65)
	for i := range values {
		values[i] = i
	}
	var calls atomic.Int64
	mapped := NewLazyMappedSeq(FnFunc1(func(value any) any {
		calls.Add(1)
		return value
	}), NewVector(values...))
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if Count(mapped) != 65 {
				t.Error("wrong concurrent chunk count")
			}
			mapped.(Hasher).Hash()
			mapped.(IHashEq).HashEq()
		}()
	}
	workers.Wait()
	if calls.Load() != 65 {
		t.Fatalf("concurrent chunk mapping ran %d times", calls.Load())
	}
}

func TestMappedSeqConvertsDirectlyToStringSlice(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any {
		calls++
		return value.(string) + "!"
	}), NewVector("a", "b", "c").Seq())

	got := asStringSlice(mapped)
	if want := []string{"a!", "b!", "c!"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mapped strings = %v, want %v", got, want)
	}
	if calls != 3 {
		t.Fatalf("mapping ran %d times, want 3", calls)
	}
	if got := mapped.First(); got != "a!" {
		t.Fatalf("cached first mapped value = %v, want a!", got)
	}
	if calls != 3 {
		t.Fatalf("cached first value ran mapping again: %d calls", calls)
	}
}

func TestMappedSeqTraversalsShareRealization(t *testing.T) {
	sum := FnFunc2(func(a, b any) any { return a.(int) + b.(int) })
	for name, visit := range map[string]func(ISeq){
		"count": func(s ISeq) {
			if Count(s) != 3 {
				t.Fatal("wrong count")
			}
		},
		"slice": func(s ISeq) {
			if len(ToSlice(s)) != 3 {
				t.Fatal("wrong slice")
			}
		},
		"vector": func(s ISeq) {
			if CreateLazilyPersistentVector(s).Count() != 3 {
				t.Fatal("wrong vector")
			}
		},
		"reduce": func(s ISeq) {
			if s.(IReduce).Reduce(sum) != 6 {
				t.Fatal("wrong sum")
			}
		},
		"reduce-init": func(s ISeq) {
			if s.(IReduceInit).ReduceInit(sum, 10) != 16 {
				t.Fatal("wrong sum")
			}
		},
		"metadata": func(s ISeq) { Count(s.(IObj).WithMeta(NewMap(NewKeyword("view"), true))) },
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			mapped := NewMappedSeq(FnFunc1(func(value any) any { calls++; return value }), NewList(1, 2, 3))
			visit(mapped)
			visit(mapped)
			ToSlice(mapped)
			if calls != 3 {
				t.Fatalf("mapper ran %d times, want 3", calls)
			}
		})
	}
}

func TestMappedSeqStringConversionsShareValues(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any { calls++; return value }), NewList("a", "b", "c"))
	asStringSlice(mapped)
	asStringSlice(mapped)
	ToSlice(mapped)
	if calls != 3 {
		t.Fatalf("string mapping ran %d times", calls)
	}
}

func TestMappedSeqEmptyTailAndNilValue(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(any) any { calls++; return nil }), NewList(1))
	if mapped.Seq() == nil || mapped.First() != nil {
		t.Fatal("nil mapped value lost its element")
	}
	rest := mapped.More()
	if rest.Seq() != nil || rest.First() != nil || Count(rest) != 0 || mapped.Next() != nil {
		t.Fatal("empty deferred tail produced an element")
	}
	if calls != 1 {
		t.Fatalf("mapper called on empty tail: %d calls", calls)
	}
	zero := FnFunc0(func() any { return "empty" })
	if rest.(IReduce).Reduce(zero) != "empty" {
		t.Fatal("empty reduction did not call zero arity")
	}
}

func TestMappedSeqRealizationRetriesAfterPanic(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		calls := 0
		failed := false
		mapped := NewMappedSeq(FnFunc1(func(value any) any {
			calls++
			if value == failAt && !failed {
				failed = true
				panic("retry")
			}
			return value
		}), NewList(1, 2))
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing mapper panic")
				}
			}()
			ToSlice(mapped)
		}()
		if !reflect.DeepEqual(ToSlice(mapped), []any{1, 2}) || calls != 3 {
			t.Fatalf("retry at %d: %d mapper calls", failAt, calls)
		}
	}
}

func TestMappedSeqReductionStopsBeforeNextMapper(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any { calls++; return value }), NewList(1, 2, 3))
	stop := FnFunc2(func(a, b any) any { return NewReduced(b) })
	if mapped.(IReduceInit).ReduceInit(stop, 0) != 1 || calls != 1 {
		t.Fatalf("reduction forced another mapper: %d calls", calls)
	}
}

func TestMappedSeqConcurrentTraversals(t *testing.T) {
	var calls atomic.Int64
	mapped := NewMappedSeq(FnFunc1(func(value any) any { calls.Add(1); return value }), NewList(1, 2, 3))
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if Count(mapped) != 3 {
				t.Error("wrong concurrent count")
			}
			mapped.(Hasher).Hash()
			mapped.(IHashEq).HashEq()
		}()
	}
	workers.Wait()
	if calls.Load() != 3 {
		t.Fatalf("concurrent mapping ran %d times", calls.Load())
	}
}

func TestMappedSeqRestKeepsInputTailLazy(t *testing.T) {
	inputCalls, mapCalls := 0, 0
	input := NewCons(1, NewLazySeq(func() any {
		inputCalls++
		return NewList(2)
	}))
	mapped := NewMappedSeq(FnFunc1(func(value any) any {
		mapCalls++
		return value
	}), input)
	rest := mapped.More()
	if inputCalls != 0 || mapCalls != 1 {
		t.Fatalf("rest forced input: input=%d, map=%d", inputCalls, mapCalls)
	}
	if mapped.Next() != rest || inputCalls != 1 || mapCalls != 2 {
		t.Fatalf("next failed to realize input: input=%d, map=%d", inputCalls, mapCalls)
	}
}
