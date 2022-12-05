package util

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/value"
)

func Seq(v interface{}) value.ISeq {
	switch v := v.(type) {
	case value.ISeq:
		return v
	case value.ISeqable:
		return v.Seq()
	}
	if reflect.TypeOf(v).Kind() == reflect.Slice {
		return value.NewSliceIterator(v)
	}
	panic(fmt.Errorf("can't convert %T to ISeq", v))
}
