package lang

import (
	"fmt"
	"reflect"
)

// Nther is an interface for compound values whose elements can be
// accessed by index.
type Nther interface {
	Nth(int) (v interface{}, ok bool)
}

// MustNth returns the nth element of the vector. It panics if the
// index is out of range.
func MustNth(x interface{}, i int) interface{} {
	v, ok := Nth(x, i)
	if !ok {
		panic("index out of range")
	}
	return v
}

func Nth(x interface{}, n int) (interface{}, bool) {
	switch x := x.(type) {
	case Nther:
		return x.Nth(n)
	case ISeq:
		x = Seq(x)
		for i := 0; i <= n; i++ {
			if x == nil {
				return nil, false
			}
			if i == n {
				return x.First(), true
			}
			x = x.Next()
		}
	case string:
		if n < 0 || n >= len(x) {
			return nil, false
		}
		return NewChar([]rune(x)[n]), true
	}

	if seq := Seq(x); seq != nil {
		return Nth(seq, n)
	}

	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if n < 0 || n >= reflectVal.Len() {
			return nil, false
		}
		return reflectVal.Index(n).Interface(), true
	}

	return nil, false
}

// NewIterator returns a lazy sequence of x, f(x), f(f(x)), ....
func NewIterator(f func(interface{}) interface{}, x interface{}) ISeq {
	return iterator{f: f, x: x}
}

type iterator struct {
	f func(interface{}) interface{}
	x interface{}
}

func (i iterator) xxx_sequential() {}

func (i iterator) Seq() ISeq {
	return i
}

func (i iterator) First() interface{} {
	return i.x
}

func (i iterator) Next() ISeq {
	return NewIterator(i.f, i.f(i.x))
}

func (i iterator) More() ISeq {
	return i.Next()
}

// NewRangeIterator returns a lazy sequence of start, start + step, start + 2*step, ...
func NewRangeIterator(start, end, step int64) ISeq {
	if end <= start {
		return emptyList
	}

	return rangeIterator{start: start, end: end, step: step}
}

type rangeIterator struct {
	// TODO: support arbitrary numeric types!
	start, end, step int64
}

var (
	_ ISeq        = (*rangeIterator)(nil)
	_ Sequential  = (*rangeIterator)(nil)
	_ IReduce     = (*rangeIterator)(nil)
	_ IReduceInit = (*rangeIterator)(nil)
)

func (r rangeIterator) xxx_sequential() {}

func (r rangeIterator) Seq() ISeq {
	return r
}

func (r rangeIterator) First() interface{} {
	return r.start
}

func (r rangeIterator) Next() ISeq {
	next := r.start + r.step
	if next >= r.end {
		return nil
	}
	return &rangeIterator{start: next, end: r.end, step: r.step}
}

func (r rangeIterator) More() ISeq {
	nxt := r.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (r rangeIterator) Reduce(f IFn) interface{} {
	var ret interface{} = r.start
	for i := r.start + r.step; i < r.end; i += r.step {
		ret = f.Invoke(ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (r rangeIterator) ReduceInit(f IFn, start interface{}) interface{} {
	var ret interface{} = start
	for i := r.start; i < r.end; i += r.step {
		ret = f.Invoke(ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

// NewValueIterator returns a lazy sequence of the values of x.
func NewVectorIterator(x IPersistentVector, start, step int) ISeq {
	if x.Count() == 0 {
		return emptyList
	}
	return vectorIterator{v: x, start: start, step: step}
}

type vectorIterator struct {
	v     IPersistentVector
	start int
	step  int
}

func (it vectorIterator) xxx_sequential() {}

func (it vectorIterator) Seq() ISeq {
	return it
}

func (it vectorIterator) First() interface{} {
	v, _ := it.v.Nth(it.start)
	return v
}

func (it vectorIterator) Next() ISeq {
	next := it.start + it.step
	if next >= it.v.Count() || next < 0 {
		return nil
	}
	return &vectorIterator{v: it.v, start: next, step: it.step}
}

func (it vectorIterator) More() ISeq {
	nxt := it.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

// NewConcatIterator returns a sequence concatenating the given
// sequences.
func NewConcatIterator(colls ...interface{}) ISeq {
	var it *concatIterator
	for i := len(colls) - 1; i >= 0; i-- {
		iseq := Seq(colls[i])
		if iseq == nil {
			continue
		}
		it = &concatIterator{seq: iseq, next: it}
	}
	if it == nil {
		return emptyList
	}
	return it
}

type concatIterator struct {
	seq  ISeq
	next *concatIterator
}

func (i *concatIterator) xxx_sequential() {}

func (i *concatIterator) Seq() ISeq {
	return i
}

func (i *concatIterator) First() interface{} {
	return i.seq.First()
}

func (i *concatIterator) Next() ISeq {
	i = &concatIterator{seq: i.seq.Next(), next: i.next}
	for i.seq == nil {
		i = i.next
		if i == nil {
			return nil
		}
	}
	return i
}

func (i *concatIterator) More() ISeq {
	nxt := i.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

////////////////////////////////////////////////////////////////////////////////

// TODO: rename to SliceSeq.

// NewSliceIterator returns a lazy sequence of the values of x.
func NewSliceIterator(x interface{}) ISeq {
	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if reflectVal.Len() == 0 {
			return nil
		}
		return sliceIterator{v: reflectVal, i: 0}
	}
	panic(fmt.Sprintf("not a slice: %T", x))
}

type sliceIterator struct {
	v reflect.Value
	i int
}

var (
	_ ISeq        = (*sliceIterator)(nil)
	_ Sequential  = (*sliceIterator)(nil)
	_ IReduce     = (*sliceIterator)(nil)
	_ IReduceInit = (*sliceIterator)(nil)
)

func (i sliceIterator) xxx_sequential() {}

func (i sliceIterator) Seq() ISeq {
	return i
}

func (i sliceIterator) First() interface{} {
	return i.v.Index(i.i).Interface()
}

func (i sliceIterator) Next() ISeq {
	i.i++
	if i.i >= i.v.Len() {
		return nil
	}
	return i
}

func (i sliceIterator) More() ISeq {
	nxt := i.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (i sliceIterator) Reduce(f IFn) interface{} {
	if i.v.IsZero() || i.v.IsNil() {
		return nil
	}

	ret := i.v.Index(i.i).Interface()
	for x := i.i + 1; x < i.v.Len(); x++ {
		ret = f.Invoke(ret, i.v.Index(x).Interface())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (i sliceIterator) ReduceInit(f IFn, start interface{}) interface{} {
	if i.v.IsZero() || i.v.IsNil() {
		return start
	}

	ret := f.Invoke(start, i.v.Index(i.i).Interface())
	for x := i.i + 1; x < i.v.Len(); x++ {
		ret = f.Invoke(ret, i.v.Index(x).Interface())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	if IsReduced(ret) {
		return ret.(IDeref).Deref()
	}
	return ret
}