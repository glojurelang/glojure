// Package arrays exposes common java.util.Arrays operations.
package arrays

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type Arrays struct{}

func Equals(left, right any) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	lv, rv := reflect.ValueOf(left), reflect.ValueOf(right)
	if !isArray(lv.Kind()) || !isArray(rv.Kind()) || lv.Len() != rv.Len() {
		return false
	}
	for index := 0; index < lv.Len(); index++ {
		l, r := lv.Index(index).Interface(), rv.Index(index).Interface()
		if isByteValue(l) && isByteValue(r) {
			if byte(lang.MustAsInt(l)) != byte(lang.MustAsInt(r)) {
				return false
			}
		} else if !lang.Equals(l, r) {
			return false
		}
	}
	return true
}

func Link() {}

func init() {
	pkgmap.SetHostClassPackage("Arrays", "java.util")
	pkgmap.SetHostClass("Arrays",
		lang.NewClass(reflect.TypeOf(Arrays{}), "java.util.Arrays"))
	fn := lang.FnFunc2(func(left, right any) any { return Equals(left, right) })
	pkgmap.Set("Arrays.equals", fn)
	pkgmap.Set("java.util.Arrays.equals", fn)
}

func isArray(kind reflect.Kind) bool {
	return kind == reflect.Array || kind == reflect.Slice
}

func isByteValue(value any) bool {
	switch value.(type) {
	case byte, int8:
		return true
	default:
		return false
	}
}
