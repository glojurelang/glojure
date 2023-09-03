package lang

import (
	"fmt"
	"reflect"
)

func SliceSet(slc any, idx int, val any) {
	slcVal := reflect.ValueOf(slc)
	slcVal.Index(idx).Set(reflect.ValueOf(val))
}

func ToSlice(x any) []any {
	if IsNil(x) {
		return nil
	}
	if s, ok := x.(ISeq); ok {
		res := make([]interface{}, 0, Count(x))
		for s := Seq(s); s != nil; s = s.Next() {
			res = append(res, s.First())
		}
		return res
	}
	xVal := reflect.ValueOf(x)
	if xVal.Kind() == reflect.Slice || xVal.Kind() == reflect.Array {
		res := make([]interface{}, xVal.Len())
		for i := 0; i < xVal.Len(); i++ {
			res[i] = xVal.Index(i).Interface()
		}
		return res
	}
	panic(fmt.Errorf("ToSlice not supported on type: %T", x))
}
