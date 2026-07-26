package lang

import (
	"sync"
	"sync/atomic"
)

// mappedSeq represents lazy map over an already-materialized, non-chunked
// sequence. It combines the lazy thunk and resulting cons cell into one object
// per element while retaining cached, thread-safe realization.
type mappedSeq struct {
	meta         IPersistentMap
	hash, hasheq uint32

	fn     any
	source ISeq

	valueOnce sync.Once
	value     any
	realized  atomic.Bool
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

func (s *mappedSeq) realizeValue() any {
	s.valueOnce.Do(func() {
		s.value = Apply1(s.fn, s.source.First())
		s.realized.Store(true)
	})
	return s.value
}

func (s *mappedSeq) xxx_sequential() {}

func (s *mappedSeq) Seq() ISeq {
	s.realizeValue()
	return s
}

func (s *mappedSeq) First() any {
	return s.realizeValue()
}

func (s *mappedSeq) Next() ISeq {
	s.realizeValue()
	return NewMappedSeq(s.fn, s.source.Next())
}

func (s *mappedSeq) More() ISeq {
	return aseqMore(s)
}

func (s *mappedSeq) Count() int {
	return aseqCount(s)
}

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
	return aseqHash(&s.hash, s)
}

func (s *mappedSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *mappedSeq) String() string {
	return aseqString(s)
}

func (s *mappedSeq) IsRealized() bool {
	return s.realized.Load()
}

func (s *mappedSeq) asStringSlice() []string {
	capacity := 0
	if source, ok := s.source.(Counted); ok {
		capacity = source.Count()
	}
	result := make([]string, 0, capacity)
	result = append(result, s.First().(string))
	for source := s.source.Next(); source != nil; source = source.Next() {
		result = append(result, Apply1(s.fn, source.First()).(string))
	}
	return result
}

func (s *mappedSeq) Reduce(f IFn) any {
	result := s.First()
	for source := s.source.Next(); source != nil; source = source.Next() {
		result = Apply2(f, result, Apply1(s.fn, source.First()))
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}

func (s *mappedSeq) ReduceInit(f IFn, init any) any {
	result := init
	for source := s.source; source != nil; source = source.Next() {
		result = Apply2(f, result, Apply1(s.fn, source.First()))
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}
