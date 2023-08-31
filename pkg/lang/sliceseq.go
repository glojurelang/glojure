package lang

import (
	"fmt"
	"reflect"
)

type (
	// SliceSeq is an implementation of ISeq for slices.
	SliceSeq struct {
		v reflect.Value
		i int
	}
)

var (
	_ ASeq = (*SliceSeq)(nil)
)

func NewSliceSeq(x interface{}) ISeq {
	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if reflectVal.Len() == 0 {
			return nil
		}
		return sliceIterator{v: reflectVal, i: 0}
	}
	panic(NewInvalidArgumentError(fmt.Sprintf("not a slice: %T", x)))
}

// // TODO: rename to SliceSeq.

// // NewSliceIterator returns a lazy sequence of the values of x.
// func NewSliceIterator(x interface{}) ISeq {
// 	reflectVal := reflect.ValueOf(x)
// 	switch reflectVal.Kind() {
// 	case reflect.Array, reflect.Slice:
// 		if reflectVal.Len() == 0 {
// 			return nil
// 		}
// 		return sliceIterator{v: reflectVal, i: 0}
// 	}
// 	panic(fmt.Sprintf("not a slice: %T", x))
// }

// type sliceIterator struct {
// 	v reflect.Value
// 	i int
// }

// var (
// 	_ ISeq        = (*sliceIterator)(nil)
// 	_ Sequential  = (*sliceIterator)(nil)
// 	_ IReduce     = (*sliceIterator)(nil)
// 	_ IReduceInit = (*sliceIterator)(nil)
// )

// func (i sliceIterator) xxx_sequential() {}

// func (i sliceIterator) Seq() ISeq {
// 	return i
// }

// func (i sliceIterator) First() interface{} {
// 	return i.v.Index(i.i).Interface()
// }

// func (i sliceIterator) Next() ISeq {
// 	i.i++
// 	if i.i >= i.v.Len() {
// 		return nil
// 	}
// 	return i
// }

// func (i sliceIterator) More() ISeq {
// 	nxt := i.Next()
// 	if nxt == nil {
// 		return emptyList
// 	}
// 	return nxt
// }

// func (i sliceIterator) Reduce(f IFn) interface{} {
// 	if i.v.IsZero() || i.v.IsNil() {
// 		return nil
// 	}

// 	ret := i.v.Index(i.i).Interface()
// 	for x := i.i + 1; x < i.v.Len(); x++ {
// 		ret = f.Invoke(ret, i.v.Index(x).Interface())
// 		if IsReduced(ret) {
// 			return ret.(IDeref).Deref()
// 		}
// 	}
// 	return ret
// }

// func (i sliceIterator) ReduceInit(f IFn, start interface{}) interface{} {
// 	if i.v.IsZero() || i.v.IsNil() {
// 		return start
// 	}

// 	ret := f.Invoke(start, i.v.Index(i.i).Interface())
// 	for x := i.i + 1; x < i.v.Len(); x++ {
// 		ret = f.Invoke(ret, i.v.Index(x).Interface())
// 		if IsReduced(ret) {
// 			return ret.(IDeref).Deref()
// 		}
// 	}
// 	if IsReduced(ret) {
// 		return ret.(IDeref).Deref()
// 	}
// 	return ret
// }
