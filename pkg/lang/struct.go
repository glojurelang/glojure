package lang

import (
	"fmt"
	"reflect"
	"sync"
	"unicode"
)

type fomKey struct {
	ptr  uintptr
	name string
}

var fomCache sync.Map // fomKey -> interface{}

// FieldOrMethod returns the field or method of the given name on the
// given value or pointer to a value, and a boolean indicating whether
// the field or method was found. If the given value is a pointer, it
// is dereferenced. If the value or pointer target is not a struct, or
// if no such field or method exists, nil and false are returned. The
// first letter of the name will be capitalized if it is not
// already. This is because Go exports fields and methods that start
// with a capital letter.
//
// Method results are cached and wrapped as FnFunc so that subsequent
// Apply calls use the IFn fast path instead of reflection.
func FieldOrMethod(v interface{}, name string) (interface{}, bool) {
	if unicode.IsLower(rune(name[0])) {
		name = string(unicode.ToUpper(rune(name[0]))) + string([]rune(name)[1:])
	}

	target := reflect.ValueOf(v)

	if !target.IsValid() {
		panic(fmt.Errorf("FieldOrMethod on nil value. field: %v", name))
	}

	// Cache for kinds that support Pointer() (ptr, func, map, slice, chan).
	// Struct values can't use Pointer(), so we skip caching for those
	// but still wrap methods as FnFunc.
	canCache := false
	var key fomKey
	switch target.Kind() {
	case reflect.Ptr, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan, reflect.UnsafePointer:
		canCache = true
		key = fomKey{target.Pointer(), name}
		if cached, ok := fomCache.Load(key); ok {
			return cached, true
		}
	}

	val := target.MethodByName(name)
	if val.IsValid() {
		result := wrapGoFunc(val.Interface())
		if canCache {
			fomCache.Store(key, result)
		}
		return result, true
	}

	// dereference the value if it's a pointer
	for target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	if target.Kind() != reflect.Struct {
		return nil, false
	}

	val = target.FieldByName(name)
	if val.IsValid() {
		return val.Interface(), true
	}

	return nil, false
}

// wrapGoFunc wraps a Go function value as FnFunc so that Apply uses
// the IFn fast path. The wrapper still uses reflect.Value.Call
// internally (unavoidable without codegen), but eliminates Apply's
// redundant coerceGoValue loop.
func wrapGoFunc(fn interface{}) FnFunc {
	goVal := reflect.ValueOf(fn)
	goType := goVal.Type()
	numIn := goType.NumIn()
	isVariadic := goType.IsVariadic()

	return FnFunc(func(args ...any) any {
		goArgs := make([]reflect.Value, len(args))
		for i, arg := range args {
			var targetType reflect.Type
			if i < numIn-1 || !isVariadic {
				if i < numIn {
					targetType = goType.In(i)
				} else {
					// Extra args beyond declared params for non-variadic
					// functions — let reflect.Call panic naturally.
					goArgs[i] = reflect.ValueOf(arg)
					continue
				}
			} else {
				targetType = goType.In(numIn - 1).Elem()
			}
			coerced, err := coerceGoValue(targetType, arg)
			if err != nil {
				panic(fmt.Errorf("arg %d: %s", i, err))
			}
			goArgs[i] = coerced
		}
		results := goVal.Call(goArgs)
		if len(results) == 0 {
			return nil
		}
		if len(results) == 1 {
			return results[0].Interface()
		}
		res := make([]interface{}, len(results))
		for i, v := range results {
			res[i] = v.Interface()
		}
		return NewVector(res...)
	})
}

func SetField(target interface{}, name string, val interface{}) error {
	targetVal := reflect.ValueOf(target)

	// dereference the value if it's a pointer
	for targetVal.Kind() == reflect.Ptr {
		targetVal = targetVal.Elem()
	}

	if targetVal.Kind() != reflect.Struct {
		return fmt.Errorf("cannot set field on non-struct")
	}

	field := targetVal.FieldByName(name)
	if field.IsValid() {
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", name)
		}
		goVal := reflect.ValueOf(val)
		if !goVal.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("cannot assign %s to %s", goVal.Type(), field.Type())
		}
		field.Set(goVal)
		return nil
	}

	return fmt.Errorf("no such field %s", name)
}
