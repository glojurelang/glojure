package value

import (
	"errors"
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

func (i rangeIterator) xxx_sequential() {}

func (i rangeIterator) Seq() ISeq {
	return i
}

func (i rangeIterator) First() interface{} {
	return i.start
}

func (i rangeIterator) Next() ISeq {
	next := i.start + i.step
	if next >= i.end {
		return nil
	}
	return &rangeIterator{start: next, end: i.end, step: i.step}
}

func (i rangeIterator) More() ISeq {
	nxt := i.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
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

type Repeat struct {
	x     interface{}
	count int64
	next  ISeq
}

var (
	_ ISeq       = (*Repeat)(nil)
	_ Sequential = (*Repeat)(nil)
)

func NewRepeat(x interface{}) *Repeat {
	return &Repeat{x: x, count: -1}
}

func NewRepeatN(count int64, x interface{}) ISeq {
	if count <= 0 {
		return emptyList
	}
	return &Repeat{x: x, count: count}
}

func (r *Repeat) xxx_sequential() {}

func (r *Repeat) First() interface{} {
	return r.x
}

func (r *Repeat) More() ISeq {
	s := r.Next()
	if s == nil {
		return emptyList
	}
	return s
}

func (r *Repeat) Next() ISeq {
	if r.next != nil {
		return r.next
	}
	if r.count > 1 {
		r.next = NewRepeatN(r.count-1, r.x)
	} else if r.count == -1 {
		r.next = r
	}
	return r.next
}

func (r *Repeat) Seq() ISeq {
	return r
}

func newIteratorError() error {
	return errors.New("iterator reached the end of collection")
}
