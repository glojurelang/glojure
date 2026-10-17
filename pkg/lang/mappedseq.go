package lang

import (
	"sync"
	"sync/atomic"
)

const (
	mappedReady uint32 = 1 << iota
	mappedNonempty
	mappedRestReady
)

// mappedSeq combines a lazy mapped value and its cached tail in one object.
// Deferred tails can be empty and are normalized by Seq before traversal.
type mappedSeq struct {
	meta         IPersistentMap
	hash, hasheq atomic.Uint32

	fn     any
	source ISeq
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
		value = Apply1(s.fn, source.First())
		state |= mappedNonempty
	}
	s.source, s.value = source, value
	if source == nil {
		s.fn = nil
	}
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
	return s
}

func (s *mappedSeq) First() any {
	s.Seq()
	return s.value
}

func (s *mappedSeq) rest() *mappedSeq {
	if s.Seq() == nil {
		return nil
	}
	if s.state.Load()&mappedRestReady != 0 {
		return s.tail
	}
	return s.realizeRest()
}

func (s *mappedSeq) realizeRest() *mappedSeq {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state.Load()
	if state&mappedRestReady == 0 {
		// More preserves a deferred input tail. Its Seq and the next mapper
		// run only when the returned mapped tail is itself realized.
		s.tail = &mappedSeq{fn: s.fn, source: s.source.More()}
		s.fn, s.source = nil, nil
		s.state.Store(state | mappedRestReady)
	}
	return s.tail
}

func (s *mappedSeq) Next() ISeq {
	if rest := s.rest(); rest != nil {
		return rest.Seq()
	}
	return nil
}

func (s *mappedSeq) More() ISeq {
	if rest := s.rest(); rest != nil {
		return rest
	}
	return emptyList
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
	return s.meta
}

func (s *mappedSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
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
	result := seq.First()
	for seq = seq.Next(); seq != nil; seq = seq.Next() {
		result = Apply2(f, result, seq.First())
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}

func (s *mappedSeq) ReduceInit(f IFn, init any) any {
	result := init
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		result = Apply2(f, result, seq.First())
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}
