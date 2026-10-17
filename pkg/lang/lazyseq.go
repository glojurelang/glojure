package lang

import (
	"sync"
	"sync/atomic"
)

type LazySeq struct {
	attrs atomic.Pointer[seqAttributes]

	fn    func() interface{}
	value any

	mu       sync.Mutex
	seqReady atomic.Bool
	producer bool
}

// LazySeqProducer lets a runtime operation co-allocate its lazy sequence
// and deferred state without a separate closure capturing that state.
type LazySeqProducer interface{ RealizeSeq() any }

// InitProducer initializes an unpublished, zero-value sequence. Producers
// may embed LazySeq so both the sequence and its state share one allocation.
func (s *LazySeq) InitProducer(producer LazySeqProducer) {
	s.value = producer
	s.producer = true
}

func NewLazySeq(fn func() interface{}) ISeq {
	return &LazySeq{fn: fn}
}

func newLazySeqWithMeta(meta IPersistentMap, seq ISeq) ISeq {
	s := &LazySeq{
		value: seq,
	}
	if meta != nil {
		s.attrs.Store(&seqAttributes{meta: meta})
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
	return !s.producer && IsNil(s.fn)
}

func (s *LazySeq) realize() interface{} {
	if s.seqReady.Load() {
		return s.readySeq()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realizeLocked()
}

func (s *LazySeq) realizeLocked() interface{} {
	if s.producer {
		s.value = s.value.(LazySeqProducer).RealizeSeq()
		s.producer = false
	} else if s.fn != nil {
		s.value = s.fn()
		s.fn = nil
	}
	return s.value
}

// A published value is either a normalized sequence or nil. The atomic
// ready flag publishes the interface words together for concurrent readers.
func (s *LazySeq) readySeq() ISeq {
	if s.value == nil {
		return nil
	}
	return s.value.(ISeq)
}

func (s *LazySeq) Seq() ISeq {
	// Publish both nonempty and empty results once. Readers can then skip
	// locking while the immutable sequence remains safely visible.
	if s.seqReady.Load() {
		return s.readySeq()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seqReady.Load() {
		return s.readySeq()
	}

	s.realizeLocked()

	if IsNil(s.value) {
		s.value = nil
		s.seqReady.Store(true)
		return s.readySeq()
	}
	ls := s.value
	for {
		lseq, ok := ls.(*LazySeq)
		if !ok {
			break
		}
		ls = lseq.realize()
	}
	s.value = Seq(ls)
	s.seqReady.Store(true)
	return s.readySeq()
}

func (s *LazySeq) Count() int {
	return Count(s.Seq())
}

func (s *LazySeq) Reduce(f IFn) interface{} {
	seq := s.Seq()
	if seq == nil {
		return Apply0(f)
	}
	initial := seq.First()
	return reduceSeqInit(seq.Next(), f, initial)
}

func (s *LazySeq) ReduceInit(f IFn, init interface{}) interface{} {
	return reduceSeqInit(s.Seq(), f, init)
}

func (s *LazySeq) Meta() IPersistentMap {
	return sequenceMeta(&s.attrs)
}

func (s *LazySeq) WithMeta(meta IPersistentMap) interface{} {
	if s.Meta() == meta {
		return s
	}

	return newLazySeqWithMeta(meta, s.Seq())
}

func (s *LazySeq) Hash() uint32 {
	return sequenceHash(&s.attrs, s, false)
}

func (s *LazySeq) HashEq() uint32 {
	return sequenceHash(&s.attrs, s, true)
}

func (s *LazySeq) String() string {
	return aseqString(s)
}
