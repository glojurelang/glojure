package value

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
func MustNth(nth Nther, i int) interface{} {
	v, ok := nth.Nth(i)
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
		for i := 0; i <= n; i++ {
			if x.IsEmpty() {
				return nil, false
			}
			if i == n {
				return x.First(), true
			}
			x = x.Rest()
		}
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
	return &iterator{f: f, x: x}
}

type iterator struct {
	f func(interface{}) interface{}
	x interface{}
}

func (i *iterator) First() interface{} {
	return i.x
}

func (i *iterator) Rest() ISeq {
	return NewIterator(i.f, i.f(i.x))
}

func (i *iterator) IsEmpty() bool {
	return false
}

// NewRangeIterator returns a lazy sequence of start, start + step, start + 2*step, ...
func NewRangeIterator(start, end, step int64) ISeq {
	if end <= start {
		return emptyList
	}

	return &rangeIterator{start: start, end: end, step: step}
}

type rangeIterator struct {
	// TODO: support arbitrary numeric types!
	start, end, step int64
}

func (i *rangeIterator) First() interface{} {
	return i.start
}

func (i *rangeIterator) Rest() ISeq {
	next := i.start + i.step
	if next >= i.end {
		return emptyList
	}
	return &rangeIterator{start: next, end: i.end, step: i.step}
}

func (i *rangeIterator) IsEmpty() bool {
	return false
}

// NewValueIterator returns a lazy sequence of the values of x.
func NewVectorIterator(x *Vector, i int) ISeq {
	return &vectorIterator{v: x, i: i}
}

type vectorIterator struct {
	v *Vector
	i int
}

func (i *vectorIterator) First() interface{} {
	return i.v.ValueAt(i.i)
}

func (i *vectorIterator) Rest() ISeq {
	return &vectorIterator{v: i.v, i: i.i + 1}
}

func (i *vectorIterator) IsEmpty() bool {
	return i.i >= i.v.Count()
}

// NewConcatIterator returns a lazy sequence of the values of x.
func NewConcatIterator(colls ...interface{}) ISeq {
	var it *concatIterator
	for i := len(colls) - 1; i >= 0; i-- {
		iseq := Seq(colls[i])
		if iseq == nil {
			panic(fmt.Sprintf("not a sequence: %T", colls[i]))
		}
		if iseq.IsEmpty() {
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

func (i *concatIterator) First() interface{} {
	return i.seq.First()
}

func (i *concatIterator) Rest() ISeq {
	i = &concatIterator{seq: i.seq.Rest(), next: i.next}
	for i.seq.IsEmpty() {
		i = i.next
		if i == nil {
			return emptyList
		}
	}
	return i
}

func (i *concatIterator) IsEmpty() bool {
	// by definition, a concat iterator is never empty
	return false
}

// NewSliceIterator returns a lazy sequence of the values of x.
func NewSliceIterator(x interface{}) ISeq {
	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if reflectVal.Len() == 0 {
			return emptyList
		}
		return sliceIterator{v: reflectVal, i: 0}
	}
	panic(fmt.Sprintf("not a slice: %T", x))
}

type sliceIterator struct {
	v reflect.Value
	i int
}

func (i sliceIterator) First() interface{} {
	return i.v.Index(i.i).Interface()
}

func (i sliceIterator) Rest() ISeq {
	i.i++
	if i.i >= i.v.Len() {
		return emptyList
	}
	return i
}

func (i sliceIterator) IsEmpty() bool {
	return false
}
