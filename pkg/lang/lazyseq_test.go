package lang

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLazySeq(t *testing.T) {
	ls := NewLazySeq(func() interface{} { return nil })
	// should be empty
	if Seq(ls) != nil {
		t.Fail()
	}
}

func TestLazySeqConcurrentRealization(t *testing.T) {
	for _, empty := range []bool{false, true} {
		var calls atomic.Int32
		value := NewLazySeq(func() any {
			calls.Add(1)
			if empty {
				return nil
			}
			return NewList(int64(42))
		}).(*LazySeq)
		if value.IsRealized() {
			t.Fatal("new lazy sequence is realized")
		}
		var workers sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < 16; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				for j := 0; j < 100; j++ {
					got := value.First()
					if empty && got != nil || !empty && got != int64(42) {
						t.Errorf("First = %v for empty=%v", got, empty)
					}
					if !value.IsRealized() {
						t.Error("realized value is pending")
					}
				}
			}()
		}
		close(start)
		workers.Wait()
		if got := calls.Load(); got != 1 {
			t.Fatalf("thunk ran %d times, want 1", got)
		}
	}
}

func TestLazySeqNestedAndMetadata(t *testing.T) {
	calls := 0
	inner := NewLazySeq(func() any {
		calls++
		return NewList(int64(42))
	})
	outer := NewLazySeq(func() any { return inner }).(*LazySeq)
	want := outer.Seq()
	meta := NewMap(NewKeyword("test"), true)
	tagged := outer.WithMeta(meta).(*LazySeq)
	if tagged.Meta() != meta || !tagged.IsRealized() {
		t.Fatal("metadata copy lost metadata or realized state")
	}
	for _, value := range []*LazySeq{outer, inner.(*LazySeq), tagged} {
		if value.Seq() != want || value.First() != int64(42) {
			t.Fatal("nested sequence lost its cached result")
		}
	}
	if calls != 1 {
		t.Fatalf("inner thunk ran %d times, want 1", calls)
	}
}

func TestLazySeqConcurrentNestedReaders(t *testing.T) {
	var calls atomic.Int32
	inner := NewLazySeq(func() any {
		calls.Add(1)
		return NewList(int64(42))
	})
	values := []ISeq{
		inner,
		NewLazySeq(func() any { return inner }),
		NewLazySeq(func() any { return inner }),
	}
	var workers sync.WaitGroup
	start := make(chan struct{})
	for _, value := range values {
		workers.Add(1)
		go func(value ISeq) {
			defer workers.Done()
			<-start
			if got := value.First(); got != int64(42) {
				t.Errorf("nested first = %v, want 42", got)
			}
		}(value)
	}
	close(start)
	workers.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("shared thunk ran %d times, want 1", got)
	}
}

type testSeqProducer struct {
	LazySeq
	calls        atomic.Int32
	empty, retry bool
}

func (p *testSeqProducer) RealizeSeq() any {
	calls := p.calls.Add(1)
	if p.retry && calls == 1 {
		panic("retry")
	}
	if p.empty {
		return nil
	}
	return NewList(int64(42))
}

func TestLazySeqProducerRetryAndConcurrentReaders(t *testing.T) {
	for _, empty := range []bool{false, true} {
		producer := &testSeqProducer{empty: empty, retry: true}
		producer.InitProducer(producer)
		if producer.IsRealized() {
			t.Fatal("producer realized during construction")
		}
		func() {
			defer func() {
				if recover() != "retry" {
					t.Error("producer lost its panic")
				}
			}()
			producer.First()
		}()
		if producer.IsRealized() {
			t.Fatal("failed producer was marked realized")
		}
		var workers sync.WaitGroup
		for i := 0; i < 16; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				result := producer.First()
				if empty && result != nil || !empty && result != int64(42) {
					t.Error("producer changed its realized result")
				}
			}()
		}
		workers.Wait()
		if producer.calls.Load() != 2 || !producer.IsRealized() {
			t.Fatal("producer did not cache a successful retry")
		}
	}
}
