package lang

import (
	"sync"
	"sync/atomic"
)

const (
	mappedReady uint32 = 1 << iota
	mappedNonempty
	mappedChunked
)

// mappedSeq combines a lazy mapped value and its cached tail in one object.
// Deferred tails can be empty and are normalized by Seq before traversal.
type mappedSeq struct {
	hash, hasheq atomic.Uint32

	fn     any
	source any
	mu     sync.Mutex
	state  atomic.Uint32
	value  any
	tail   *mappedSeq
}

var (
	_ ASeq        = (*mappedSeq)(nil)
	_ IPending    = (*mappedSeq)(nil)
	_ IReduce     = (*mappedSeq)(nil)
	_ IReduceInit = (*mappedSeq)(nil)
)

func NewMappedSeq(fn any, source ISeq) ISeq {
	if source == nil {
		return nil
	}
	return &mappedSeq{fn: fn, source: source}
}

// NewLazyMappedSeq accepts an unrealized collection while retaining the
// mapper and its cached values in one object, without a per-element thunk.
func NewLazyMappedSeq(fn, coll any) ISeq {
	return &mappedSeq{fn: fn, source: coll}
}

// Publish a value only after its mapper succeeds. A failed realization can
// then be retried, just as a LazySeq retains its thunk after a panic.
func (s *mappedSeq) realize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Load()&mappedReady != 0 {
		return
	}
	source := Seq(s.source)
	state := mappedReady
	var value any
	if source != nil {
		if chunked, ok := source.(IChunkedSeq); ok {
			input := chunked.ChunkedFirst()
			values := make([]any, input.Count())
			for i := range values {
				values[i] = Apply1(s.fn, input.Nth(i))
			}
			s.value = NewChunkedCons(NewSliceChunk(values),
				NewLazyMappedSeq(s.fn, chunked.ChunkedMore()))
			s.fn, s.source = nil, nil
			s.state.Store(mappedReady | mappedNonempty | mappedChunked)
			return
		}
		value = Apply1(s.fn, source.First())
		// Capture the input rest after mapping the head, without realizing
		// that rest or its mapper. Publish both together under one lock.
		s.tail = &mappedSeq{fn: s.fn, source: source.More()}
		state |= mappedNonempty
	}
	s.value = value
	s.fn, s.source = nil, nil
	s.state.Store(state)
}

func (s *mappedSeq) xxx_sequential() {}

func (s *mappedSeq) Seq() ISeq {
	state := s.state.Load()
	if state&mappedReady == 0 {
		s.realize()
		state = s.state.Load()
	}
	if state&mappedNonempty == 0 {
		return nil
	}
	if state&mappedChunked != 0 {
		return s.value.(ISeq)
	}
	return s
}

func (s *mappedSeq) First() any {
	if seq := s.Seq(); seq != nil && seq != s {
		return seq.First()
	}
	return s.value
}

func (s *mappedSeq) Next() ISeq {
	seq := s.Seq()
	if seq == nil {
		return nil
	}
	if seq != s {
		return seq.Next()
	}
	return s.tail.Seq()
}

func (s *mappedSeq) More() ISeq {
	seq := s.Seq()
	if seq == nil {
		return emptyList
	}
	if seq != s {
		return seq.More()
	}
	return s.tail
}

func (s *mappedSeq) Count() int { return Count(s) }

func (s *mappedSeq) Cons(value any) Conser {
	return aseqCons(s, value)
}

func (s *mappedSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *mappedSeq) Equals(other any) bool {
	return aseqEquals(s, other)
}

func (s *mappedSeq) Equiv(other any) bool {
	return aseqEquiv(s, other)
}

func (s *mappedSeq) Meta() IPersistentMap {
	return nil
}

func (s *mappedSeq) WithMeta(meta IPersistentMap) any {
	if meta == nil {
		return s
	}
	return newLazySeqWithMeta(meta, s.Seq())
}

func (s *mappedSeq) Hash() uint32 {
	if value := s.hash.Load(); value != 0 {
		return value
	}
	var value uint32
	aseqHash(&value, s)
	s.hash.CompareAndSwap(0, value)
	return value
}

func (s *mappedSeq) HashEq() uint32 {
	if value := s.hasheq.Load(); value != 0 {
		return value
	}
	var value uint32
	aseqHashEq(&value, s)
	s.hasheq.CompareAndSwap(0, value)
	return value
}

func (s *mappedSeq) String() string {
	return aseqString(s)
}

func (s *mappedSeq) IsRealized() bool {
	return s.state.Load()&mappedReady != 0
}

func (s *mappedSeq) asStringSlice() []string {
	result := make([]string, 0)
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		result = append(result, seq.First().(string))
	}
	return result
}

func (s *mappedSeq) Reduce(f IFn) any {
	seq := s.Seq()
	if seq == nil {
		return Apply0(f)
	}
	initial := seq.First()
	return reduceSeqInit(seq.Next(), f, initial)
}

func (s *mappedSeq) ReduceInit(f IFn, init any) any {
	return reduceSeqInit(s.Seq(), f, init)
}
