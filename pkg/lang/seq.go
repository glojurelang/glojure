package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/seq"
)

func First(x interface{}) interface{} {
	if x == nil {
		return nil
	}
	s := Seq(x)
	if s == nil {
		return nil
	}
	return s.First()
}

func Rest(x interface{}) interface{} {
	s := Seq(x)
	if s == nil {
		return emptyList
	}
	return s.More()
}

func Next(x interface{}) ISeq {
	if s, ok := x.(ISeq); ok {
		return s.Next()
	}

	s := Seq(x)
	if s == nil {
		return emptyList
	}
	return s.Next()
}

func IsSeq(x interface{}) bool {
	_, ok := x.(ISeq)
	return ok
}

func Seq(x interface{}) ISeq {
	switch x := x.(type) {
	case *EmptyList:
		return nil
	case *LazySeq:
		return x.Seq()
	case ISeq:
		return x
	case Seqable:
		return x.Seq()
	case string:
		return NewStringSeq(x, 0)
	case nil:
		return nil
		// TODO: define an Iterable interface, and use it here.
	}

	// use the reflect package to handle slices and arrays
	t := reflect.TypeOf(x)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return NewSliceSeq(x)
	case reflect.Map:
		return NewGoMapSeq(x)
	}

	panic(fmt.Errorf("can't convert %T to ISeq", x))
}

func seqToSlice(s ISeq) []interface{} {
	var res []interface{}
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		res = append(res, seq.First())
	}
	return res
}

type seqSeq struct {
	ISeq
}

func (s seqSeq) Next() seq.Seq {
	n := s.ISeq.Next()
	if n == nil {
		return nil
	}
	return seqSeq{ISeq: n}
}

func seqToInternalSeq(s ISeq) seq.Seq {
	if s == nil {
		return nil
	}
	return seqSeq{ISeq: s}
}
