package lang

import (
	"sync"
	"sync/atomic"
)

type LazySeq struct {
	meta         IPersistentMap
	hash, hasheq uint32

	fn  func() interface{}
	sv  interface{}
	seq ISeq

	mu       sync.Mutex
	seqReady atomic.Bool
}

func NewLazySeq(fn func() interface{}) ISeq {
	return &LazySeq{fn: fn}
}

func newLazySeqWithMeta(meta IPersistentMap, seq ISeq) ISeq {
	s := &LazySeq{
		meta: meta,
		seq:  seq,
	}
	s.seqReady.Store(true)
	return s
}

var (
	_ ASeq                  = (*LazySeq)(nil)
	_ ISeq                  = (*LazySeq)(nil)
	_ IPending              = (*LazySeq)(nil)
	_ IObj                  = (*LazySeq)(nil)
	_ Counter               = (*LazySeq)(nil)
	_ Sequential            = (*LazySeq)(nil)
	_ IPersistentCollection = (*LazySeq)(nil)
	_ IHashEq               = (*LazySeq)(nil)
	_ IReduce               = (*LazySeq)(nil)
	_ IReduceInit           = (*LazySeq)(nil)
)

func (s *LazySeq) xxx_sequential() {}

func (s *LazySeq) First() interface{} {
	seq := s.Seq()
	if IsNil(seq) {
		return nil
	}
	return seq.First()
}

func (s *LazySeq) Next() ISeq {
	seq := s.Seq()
	if IsNil(seq) {
		return nil
	}
	return seq.Next()
}

func (s *LazySeq) More() ISeq {
	seq := s.Seq()
	if IsNil(seq) {
		return emptyList
	}
	return seq.More()
}

func (s *LazySeq) Cons(x interface{}) Conser {
	return NewCons(x, s)
}

func (s *LazySeq) Empty() IPersistentCollection {
	return emptyList
}

func (s *LazySeq) Equals(o interface{}) bool {
	seq := s.Seq()
	if !IsNil(seq) {
		return Equals(seq, o)
	}
	return IsNil(Seq(o))
}

func (s *LazySeq) Equiv(o interface{}) bool {
	seq := s.Seq()
	if !IsNil(seq) {
		return Equiv(seq, o)
	}
	return IsNil(Seq(o))
}

func (s *LazySeq) IsRealized() bool {
	if s.seqReady.Load() {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return IsNil(s.fn)
}

func (s *LazySeq) realize() interface{} {
	if s.seqReady.Load() {
		return s.seq
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realizeLocked()
}

func (s *LazySeq) realizeLocked() interface{} {
	if s.fn != nil {
		s.sv = s.fn()
		s.fn = nil
	}
	if s.sv != nil {
		return s.sv
	}
	return s.seq
}

func (s *LazySeq) Seq() ISeq {
	// Publish both nonempty and empty results once. Readers can then skip
	// locking while the immutable sequence remains safely visible.
	if s.seqReady.Load() {
		return s.seq
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seqReady.Load() {
		return s.seq
	}

	s.realizeLocked()

	if IsNil(s.sv) {
		s.seqReady.Store(true)
		return s.seq
	}
	ls := s.sv
	s.sv = nil
	for {
		lseq, ok := ls.(*LazySeq)
		if !ok {
			break
		}
		ls = lseq.realize()
	}
	s.seq = Seq(ls)
	s.seqReady.Store(true)
	return s.seq
}

func (s *LazySeq) Count() int {
	c := 0
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		c++
	}
	return c
}

func (s *LazySeq) Reduce(f IFn) interface{} {
	seq := s.Seq()
	if seq == nil {
		return Apply0(f)
	}
	ret := seq.First()
	for seq = seq.Next(); seq != nil; seq = seq.Next() {
		ret = Apply2(f, ret, seq.First())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *LazySeq) ReduceInit(f IFn, init interface{}) interface{} {
	ret := init
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		ret = Apply2(f, ret, seq.First())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *LazySeq) Meta() IPersistentMap {
	return s.meta
}

func (s *LazySeq) WithMeta(meta IPersistentMap) interface{} {
	if s.meta == meta {
		return s
	}

	return newLazySeqWithMeta(meta, s.Seq())
}

func (s *LazySeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *LazySeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *LazySeq) String() string {
	return aseqString(s)
}
