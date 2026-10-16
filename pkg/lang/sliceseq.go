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

		v    reflect.Value
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
	return s.v.Index(s.i).Interface()
}

func (s *SliceSeq) Next() ISeq {
	nxt := s.i + 1
	if nxt >= s.v.Len() {
		return nil
	}
	cache := s.v.Len()-nxt <= sliceSeqTailCacheLimit
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
	return s.v.Len() - s.i
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
	if s.v.IsZero() || s.v.IsNil() {
		return nil
	}

	ret := s.v.Index(s.i).Interface()
	for x := s.i + 1; x < s.v.Len(); x++ {
		ret = f.Invoke(ret, s.v.Index(x).Interface())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *SliceSeq) ReduceInit(f IFn, start any) any {
	if s.v.IsZero() || s.v.IsNil() {
		return start
	}

	ret := f.Invoke(start, s.v.Index(s.i).Interface())
	for x := s.i + 1; x < s.v.Len(); x++ {
		ret = f.Invoke(ret, s.v.Index(x).Interface())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	if IsReduced(ret) {
		return ret.(IDeref).Deref()
	}
	return ret
}
