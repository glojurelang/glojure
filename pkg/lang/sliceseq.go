package lang

import (
	"fmt"
	"reflect"
)

type (
	// SliceSeq is an implementation of ISeq for slices.
	SliceSeq struct {
		meta         IPersistentMap
		hash, hasheq int32

		v reflect.Value
		i int
	}
)

var (
	_ ASeq        = (*SliceSeq)(nil)
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
		return sliceIterator{v: reflectVal, i: 0}
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

	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *SliceSeq) First() any {
	return s.v.Index(s.i).Interface()
}

func (s *SliceSeq) Next() ISeq {
	nxt := s.i + 1
	if nxt >= s.v.Len() {
		return nil
	}
	return &SliceSeq{
		v: s.v,
		i: nxt,
	}
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

func (s *SliceSeq) Empty() IPersistentCollection {
	return aseqEmpty(s)
}

func (s *SliceSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *SliceSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *SliceSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *SliceSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
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
