package lang

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

// Keep retained suffix views bounded to one ordinary sequence chunk.
const sliceSeqTailCacheLimit = 32

type (
	// SliceSeq is an implementation of ISeq for slices.
	SliceSeq struct {
		meta         IPersistentMap
		hash, hasheq atomic.Uint32

		// Ordinary language slices need no reflection. Other host slices and
		// arrays retain a reflect.Value view of their backing storage.
		v    any
		i    int
		next atomic.Pointer[SliceSeq]
	}
)

var (
	_ ASeq        = (*SliceSeq)(nil)
	_ Counted     = (*SliceSeq)(nil)
	_ IReduce     = (*SliceSeq)(nil)
	_ IReduceInit = (*SliceSeq)(nil)
)

func NewSliceSeq(x any) ISeq {
	if values, ok := x.([]any); ok {
		if len(values) == 0 {
			return nil
		}
		return &SliceSeq{v: x}
	}
	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if reflectVal.Len() == 0 {
			return nil
		}
		return &SliceSeq{v: reflectVal, i: 0}
	}
	panic(NewIllegalArgumentError(fmt.Sprintf("not a slice: %T", x)))
}

func (s *SliceSeq) xxx_sequential() {}

// HasValue distinguishes a constructed nonempty sequence from its zero value.
// Interpreter call guards use it before bypassing general panic recovery.
func (s *SliceSeq) HasValue() bool {
	// All constructors reject empty inputs and Next checks its bound before
	// creating a suffix. Only the exported zero value lacks backing storage.
	return s != nil && s.v != nil
}

func (s *SliceSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *SliceSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}

	copy := &SliceSeq{meta: meta, v: s.v, i: s.i}
	copy.hash.Store(s.hash.Load())
	copy.hasheq.Store(s.hasheq.Load())
	copy.next.Store(s.next.Load())
	return copy
}

func (s *SliceSeq) First() any {
	return s.at(s.i)
}

func (s *SliceSeq) at(i int) any {
	if values, ok := s.v.([]any); ok {
		return values[i]
	}
	return s.v.(reflect.Value).Index(i).Interface()
}

func (s *SliceSeq) length() int {
	if values, ok := s.v.([]any); ok {
		return len(values)
	}
	return s.v.(reflect.Value).Len()
}

func (s *SliceSeq) Next() ISeq {
	nxt := s.i + 1
	length := s.length()
	if nxt >= length {
		return nil
	}
	cache := length-nxt <= sliceSeqTailCacheLimit
	if cache {
		if next := s.next.Load(); next != nil {
			return next
		}
	}
	next := &SliceSeq{
		v: s.v,
		i: nxt,
	}
	if !cache || s.next.CompareAndSwap(nil, next) {
		return next
	}
	return s.next.Load()
}

func (s *SliceSeq) More() ISeq {
	return aseqMore(s)
}

func (s *SliceSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *SliceSeq) Count() int {
	return s.length() - s.i
}

func (s *SliceSeq) xxx_counted() {}

func (s *SliceSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *SliceSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *SliceSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *SliceSeq) Hash() uint32 {
	if hash := s.hash.Load(); hash != 0 {
		return hash
	}
	var hash uint32
	aseqHash(&hash, s)
	s.hash.CompareAndSwap(0, hash)
	return hash
}

func (s *SliceSeq) HashEq() uint32 {
	if hash := s.hasheq.Load(); hash != 0 {
		return hash
	}
	var hash uint32
	aseqHashEq(&hash, s)
	s.hasheq.CompareAndSwap(0, hash)
	return hash
}

func (s *SliceSeq) Seq() ISeq {
	return s
}

func (s *SliceSeq) String() string {
	return aseqString(s)
}

func (s *SliceSeq) Reduce(f IFn) any {
	if s.v == nil {
		return nil
	}

	ret := s.First()
	for x := s.i + 1; x < s.length(); x++ {
		ret = f.Invoke(ret, s.at(x))
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *SliceSeq) ReduceInit(f IFn, start any) any {
	if s.v == nil {
		return start
	}

	ret := f.Invoke(start, s.First())
	for x := s.i + 1; x < s.length(); x++ {
		ret = f.Invoke(ret, s.at(x))
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	if IsReduced(ret) {
		return ret.(IDeref).Deref()
	}
	return ret
}
