package value

import (
	"sync"
)

type LazySeq struct {
	fn  func() interface{}
	sv  interface{}
	seq ISeq

	meta IPersistentMap

	realizeMtx sync.RWMutex
	seqMtx     sync.Mutex
}

func NewLazySeq(fn func() interface{}) ISeq {
	return &LazySeq{fn: fn}
}

func newLazySeqWithMeta(meta IPersistentMap, seq ISeq) ISeq {
	return &LazySeq{
		meta: meta,
		seq:  seq,
	}
}

var (
	_ ISeq                  = (*LazySeq)(nil)
	_ IPending              = (*LazySeq)(nil)
	_ IObj                  = (*LazySeq)(nil)
	_ Counted               = (*LazySeq)(nil)
	_ Sequential            = (*LazySeq)(nil)
	_ IPersistentCollection = (*LazySeq)(nil)
	// IHashEq
)

func (s *LazySeq) xxx_sequential() {}

func (s *LazySeq) First() interface{} {
	seq := s.Seq()
	if seq == nil {
		return nil
	}
	return seq.First()
}

func (s *LazySeq) Next() ISeq {
	seq := s.Seq()
	if seq == nil {
		return nil
	}
	return seq.Next()
}

func (s *LazySeq) More() ISeq {
	seq := s.Seq()
	if seq == nil {
		return emptyList
	}
	return seq.More()
}

func (s *LazySeq) Cons(x interface{}) ISeq {
	return NewCons(x, s)
}

func (s *LazySeq) Conj(x interface{}) Conjer {
	return s.Cons(x).(Conjer)
}

func (s *LazySeq) Empty() IPersistentCollection {
	return emptyList
}

func (s *LazySeq) Equal(o interface{}) bool {
	seq := s.Seq()
	if s != nil {
		return Equal(seq, o)
	}
	return Seq(o) == nil
}

func (s *LazySeq) IsRealized() bool {
	s.realizeMtx.RLock()
	defer s.realizeMtx.RUnlock()
	return s.fn == nil
}

func (s *LazySeq) realize() interface{} {
	s.realizeMtx.Lock()
	defer s.realizeMtx.Unlock()

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
	s.seqMtx.Lock()
	defer s.seqMtx.Unlock()

	s.realize()

	if s.sv == nil {
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
	return s.seq
}

func (s *LazySeq) Count() int {
	c := 0
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		c++
	}
	return c
}

func (s *LazySeq) Meta() IPersistentMap {
	return s.meta
}

func (s *LazySeq) WithMeta(meta IPersistentMap) interface{} {
	if Equal(s.meta, meta) {
		return s
	}

	return newLazySeqWithMeta(meta, s.Seq())
}
