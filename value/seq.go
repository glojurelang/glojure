package value

import (
	"fmt"
	"reflect"
)

func First(x interface{}) interface{} {
	if x == nil {
		return nil
	}
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
	if x == nil {
		return nil
	}
	if s := Seq(x); s != nil {
		return s.Next()
	}
	panic(fmt.Errorf("%T can't be converted to ISeq", x))
}

func Seq(x interface{}) ISeq {
	switch x := x.(type) {
	case ISeq:
		if x.IsEmpty() {
			return nil
		}
		return x
	case ISeqable:
		return x.Seq()
	case string:
		return newStringSeq(x)
	case nil:
		return nil
		// TODO: define an Iterable interface, and use it here.
	}

	// use the reflect package to handle slices and arrays
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return newSliceSeq(v)
	}

	panic(fmt.Errorf("can't convert %T to ISeq", v.Interface()))
}

// TODO: deduplicate this with NewSliceIterator.
func newSliceSeq(x reflect.Value) ISeq {
	return NewSliceIterator(x)
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

func (s stringSeq) Next() ISeq {
	if s.i+1 >= len(s.v) {
		return nil
	}
	return stringSeq{v: s.v, i: s.i + 1}
}

func (s stringSeq) Rest() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (s stringSeq) IsEmpty() bool {
	// by construction, s.i is always in range, so we don't need to
	// check.
	return false
}
