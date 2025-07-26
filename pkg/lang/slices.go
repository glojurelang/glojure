package lang

import (
	"fmt"
	"reflect"
)

func SliceSet(slc any, idx int, val any) {
	slcVal := reflect.ValueOf(slc)
	if IsNil(val) {
		// Handle nil values specially
		slcVal.Index(idx).Set(reflect.Zero(slcVal.Index(idx).Type()))
	} else {
		slcVal.Index(idx).Set(reflect.ValueOf(val))
	}
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
	if v, ok := x.(*Vector); ok {
		res := make([]interface{}, v.Count())
		for i := 0; i < v.Count(); i++ {
			res[i] = v.Nth(i)
		}
		return res
	}
	if str, ok := x.(string); ok {
		res := make([]interface{}, len(str))
		for i, r := range str {
			res[i] = Char(r)
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
