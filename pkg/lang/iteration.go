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
		if seq == x {
			panic(fmt.Errorf("unexpected Seq result equal to input"))
		}
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

// // NewIterator returns a lazy sequence of x, f(x), f(f(x)), ....
// func NewIterator(f func(interface{}) interface{}, x interface{}) ISeq {
// 	return iterator{f: f, x: x}
// }

// type iterator struct {
// 	f func(interface{}) interface{}
// 	x interface{}
// }

// func (i iterator) xxx_sequential() {}

// func (i iterator) Seq() ISeq {
// 	return i
// }

// func (i iterator) First() interface{} {
// 	return i.x
// }

// func (i iterator) Next() ISeq {
// 	return NewIterator(i.f, i.f(i.x))
// }

// func (i iterator) More() ISeq {
// 	nxt := i.Next()
// 	if nxt == nil {
// 		return emptyList
// 	}
// 	return nxt
// }

// // NewValueIterator returns a lazy sequence of the values of x.
// func NewVectorIterator(x IPersistentVector, start, step int) ISeq {
// 	if x.Count() == 0 {
// 		return emptyList
// 	}
// 	return vectorIterator{v: x, start: start, step: step}
// }

// type vectorIterator struct {
// 	v     IPersistentVector
// 	start int
// 	step  int
// }

// func (it vectorIterator) xxx_sequential() {}

// func (it vectorIterator) Seq() ISeq {
// 	return it
// }

// func (it vectorIterator) First() interface{} {
// 	return it.v.Nth(it.start)
// }

// func (it vectorIterator) Next() ISeq {
// 	next := it.start + it.step
// 	if next >= it.v.Count() || next < 0 {
// 		return nil
// 	}
// 	return &vectorIterator{v: it.v, start: next, step: it.step}
// }

// func (it vectorIterator) More() ISeq {
// 	nxt := it.Next()
// 	if nxt == nil {
// 		return emptyList
// 	}
// 	return nxt
// }

// // NewConcatIterator returns a sequence concatenating the given
// // sequences.
// func NewConcatIterator(colls ...interface{}) ISeq {
// 	var it *concatIterator
// 	for i := len(colls) - 1; i >= 0; i-- {
// 		iseq := Seq(colls[i])
// 		if iseq == nil {
// 			continue
// 		}
// 		it = &concatIterator{seq: iseq, next: it}
// 	}
// 	if it == nil {
// 		return emptyList
// 	}
// 	return it
// }

// type concatIterator struct {
// 	seq  ISeq
// 	next *concatIterator
// }

// func (i *concatIterator) xxx_sequential() {}

// func (i *concatIterator) Seq() ISeq {
// 	return i
// }

// func (i *concatIterator) First() interface{} {
// 	return i.seq.First()
// }

// func (i *concatIterator) Next() ISeq {
// 	i = &concatIterator{seq: i.seq.Next(), next: i.next}
// 	for i.seq == nil {
// 		i = i.next
// 		if i == nil {
// 			return nil
// 		}
// 	}
// 	return i
// }

// func (i *concatIterator) More() ISeq {
// 	nxt := i.Next()
// 	if nxt == nil {
// 		return emptyList
// 	}
// 	return nxt
// }

// ////////////////////////////////////////////////////////////////////////////////

// func chunkIteratorSeq(iter *reflect.MapIter) ISeq {
// 	const chunkSize = 32

// 	return NewLazySeq(func() interface{} {
// 		chunk := make([]interface{}, 0, chunkSize)
// 		exhausted := false
// 		for n := 0; n < chunkSize; n++ {
// 			chunk = append(chunk, NewMapEntry(iter.Key().Interface(), iter.Value().Interface()))
// 			if !iter.Next() {
// 				exhausted = true
// 				break
// 			}
// 		}
// 		if exhausted {
// 			return NewChunkedCons(NewSliceChunk(chunk), nil)
// 		}
// 		return NewChunkedCons(NewSliceChunk(chunk), chunkIteratorSeq(iter))
// 	})
// }

// func NewGoMapSeq(x interface{}) ISeq {
// 	rng := reflect.ValueOf(x).MapRange()
// 	if !rng.Next() {
// 		return nil
// 	}
// 	return chunkIteratorSeq(rng)
// }
