package lang

import "reflect"

func SliceSet(slc interface{}, idx int, val interface{}) {
	slcVal := reflect.ValueOf(slc)
	slcVal.Index(idx).Set(reflect.ValueOf(val))
}
