package value

import (
	"fmt"
	"reflect"
)

type ISeq interface {
	First() interface{}
	Rest() ISeq
	IsEmpty() bool
}

type ISeqable interface {
	Seq() ISeq
}

func First(x interface{}) interface{} {
	if s := Seq(x); s != nil {
		return s.First()
	}
	panic(fmt.Errorf("%T can't be converted to ISeq", x))
}

func Rest(x interface{}) interface{} {
	if s := Seq(x); s != nil {
		return s.Rest()
	}
	panic(fmt.Errorf("%T can't be converted to ISeq", x))
}

func Next(x interface{}) interface{} {
	if s := Seq(x); s != nil {
		rst := s.Rest()
		if rst.IsEmpty() {
			return nil
		}
		return rst
	}
	panic(fmt.Errorf("%T can't be converted to ISeq", x))
}

func Seq(x interface{}) ISeq {
	switch x := x.(type) {
	case ISeq:
		return x
	case ISeqable:
		return x.Seq()
	case string:
		return newStringSeq(x)
	case nil:
		return emptyList
	}

	// use the reflect package to handle slices and arrays
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return newSliceSeq(v)
	}
	return nil
}

func newSliceSeq(x reflect.Value) ISeq {
	if x.Len() == 0 {
		return emptyList
	}
	return sliceSeq{v: x, i: 0}
}

type sliceSeq struct {
	v reflect.Value
	i int
}

func (s sliceSeq) First() interface{} {
	return s.v.Index(s.i).Interface()
}

func (s sliceSeq) Rest() ISeq {
	if s.i+1 >= s.v.Len() {
		return emptyList
	}
	return sliceSeq{v: s.v, i: s.i + 1}
}

func (s sliceSeq) IsEmpty() bool {
	// by construction, s.i is always in range, so we don't need to
	// check.
	return false
}

func newStringSeq(x string) ISeq {
	if x == "" {
		return emptyList
	}
	return stringSeq{v: x, i: 0}
}

type stringSeq struct {
	v string
	i int
}

func (s stringSeq) First() interface{} {
	return NewChar(rune(s.v[s.i]))
}

func (s stringSeq) Rest() ISeq {
	if s.i+1 >= len(s.v) {
		return emptyList
	}
	return stringSeq{v: s.v, i: s.i + 1}
}

func (s stringSeq) IsEmpty() bool {
	// by construction, s.i is always in range, so we don't need to
	// check.
	return false
}
