// Package arrays exposes common java.util.Arrays operations.
package arrays

import (
	"fmt"
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

func NewInstance(component any, dimensions any) any {
	typ, ok := lang.ReflectType(component)
	if !ok {
		panic(fmt.Sprintf("Array.newInstance: expected Class, got %T", component))
	}
	dims := dimensionSlice(dimensions)
	if len(dims) == 0 {
		panic("Array.newInstance: at least one dimension is required")
	}
	return newArray(typ, dims).Interface()
}

func dimensionSlice(value any) []int {
	if lang.IsNumber(value) {
		return []int{lang.MustAsInt(value)}
	}
	sequence := lang.Seq(value)
	result := make([]int, 0, lang.Count(value))
	for ; sequence != nil; sequence = sequence.Next() {
		result = append(result, lang.MustAsInt(sequence.First()))
	}
	return result
}

func newArray(component reflect.Type, dimensions []int) reflect.Value {
	if dimensions[0] < 0 {
		panic(fmt.Sprintf("Array.newInstance: negative array size %d", dimensions[0]))
	}
	if len(dimensions) == 1 {
		return reflect.MakeSlice(reflect.SliceOf(component), dimensions[0], dimensions[0])
	}
	innerType := component
	for range dimensions[1:] {
		innerType = reflect.SliceOf(innerType)
	}
	result := reflect.MakeSlice(reflect.SliceOf(innerType), dimensions[0], dimensions[0])
	for index := 0; index < result.Len(); index++ {
		result.Index(index).Set(newArray(component, dimensions[1:]))
	}
	return result
}

func Link() {}

func init() {
	pkgmap.SetHostClassPackage("Arrays", "java.util")
	pkgmap.SetHostClass("Arrays",
		lang.NewClass(reflect.TypeOf(Arrays{}), "java.util.Arrays"))
	fn := lang.FnFunc2(func(left, right any) any { return Equals(left, right) })
	pkgmap.Set("Arrays.equals", fn)
	pkgmap.Set("java.util.Arrays.equals", fn)
	newInstance := lang.FnFunc2(func(component, dimensions any) any {
		return NewInstance(component, dimensions)
	})
	pkgmap.Set("Array.newInstance", newInstance)
	pkgmap.Set("java.lang.reflect.Array.newInstance", newInstance)
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
