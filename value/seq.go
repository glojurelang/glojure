package value

import (
	"fmt"
	"reflect"
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
	case ISeqable:
		return x.Seq()
	case string:
		return newStringSeq(x)
	case nil:
		return nil
		// TODO: define an Iterable interface, and use it here.
	}

	// use the reflect package to handle slices and arrays
	t := reflect.TypeOf(x)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return NewSliceIterator(x)
	}

	panic(fmt.Errorf("can't convert %T to ISeq", x))
}

func SeqsEqual(seq1, seq2 ISeq) bool {
	for seq1 != nil {
		if seq2 == nil || !Equal(seq1.First(), seq2.First()) {
			return false
		}
		seq1 = seq1.Next()
		seq2 = seq2.Next()
	}
	return seq2 == nil
}

func IsSeqEqual(seq ISeq, other interface{}) bool {
	if seq == other {
		return true
	}
	switch other := other.(type) {
	case Sequential:
		switch other := other.(type) {
		case ISeqable:
			return SeqsEqual(seq, other.Seq())
		}
	}
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

func (s stringSeq) xxx_sequential() {}

func (s stringSeq) Seq() ISeq {
	return s
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

func (s stringSeq) More() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func seqToSlice(s ISeq) []interface{} {
	var res []interface{}
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		res = append(res, seq.First())
	}
	return res
}
