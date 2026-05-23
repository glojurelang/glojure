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

// StringMethod is the signature for JVM-style instance methods on
// java.lang.String. The receiver is passed as the first argument and any
// further arguments arrive in rest. Bridge implementations are
// responsible for argument-count validation and type coercion.
type StringMethod func(s string, rest ...any) any

var stringMethods = map[string]StringMethod{}

// RegisterStringMethod registers fn as the implementation of the given
// JVM-style method name on java.lang.String (e.g. "length",
// "toUpperCase", "substring"). Called from package init in the
// javacompat layer; not safe for concurrent use after startup.
func RegisterStringMethod(name string, fn StringMethod) {
	stringMethods[name] = fn
}

func lookupStringMethod(name string) (StringMethod, bool) {
	fn, ok := stringMethods[name]
	return fn, ok
}

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
	// Strings have no Go-level methods; dispatch JVM-style names like
	// toUpperCase, length, substring through the javacompat/string
	// registry. The lookup is case-insensitive on the first letter so
	// rewrite-core's lower-to-upper renames (e.g. .equals -> .Equals)
	// still resolve. The returned IFn captures the receiver and accepts
	// only the remaining arguments.
	if s, isStr := v.(string); isStr {
		lookup := name
		if len(lookup) > 0 && unicode.IsUpper(rune(lookup[0])) {
			lookup = string(unicode.ToLower(rune(lookup[0]))) + lookup[1:]
		}
		if fn, ok := lookupStringMethod(lookup); ok {
			return FnFunc(func(args ...any) any { return fn(s, args...) }), true
		}
	}

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

// wrapGoFunc wraps a Go function value as IFn so that Apply uses
// the IFn fast path. For common signatures, it creates a direct-call
// FnFuncN wrapper with zero allocation per call. Exotic signatures
// fall back to reflect.Value.Call wrapped as FnFunc.
func wrapGoFunc(fn interface{}) IFn {
	// Fast path: type-switch on common function signatures.
	// The type assertion happens once at wrap time; all subsequent
	// calls are direct Go function calls with no reflection.
	switch f := fn.(type) {
	// --- 0 args ---
	case func() any:
		return FnFunc0(func() any { return f() })
	case func() int:
		return FnFunc0(func() any { return f() })
	case func() bool:
		return FnFunc0(func() any { return f() })
	case func():
		return FnFunc0(func() any { f(); return nil })

	// --- 1 arg, any param ---
	case func(any) any:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any) bool:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any) int:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any) int64:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any) Char:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any):
		return FnFunc1(func(a any) any { f(a); return nil })

	// --- 1 arg, typed param ---
	case func(string) string:
		return FnFunc1(func(a any) any { return f(a.(string)) })
	case func(string):
		return FnFunc1(func(a any) any { f(a.(string)); return nil })

	// --- 2 args, all any ---
	case func(any, any) any:
		return FnFunc2(func(a, b any) any { return f(a, b) })
	case func(any, any) bool:
		return FnFunc2(func(a, b any) any { return f(a, b) })
	case func(any, any) int:
		return FnFunc2(func(a, b any) any { return f(a, b) })
	case func(any, any) int64:
		return FnFunc2(func(a, b any) any { return f(a, b) })
	case func(any, any):
		return FnFunc2(func(a, b any) any { f(a, b); return nil })

	// --- 2 args, mixed typed ---
	case func(any, int) any:
		return FnFunc2(func(a, b any) any { return f(a, MustAsInt(b)) })

	// --- 3 args ---
	case func(any, any, any) any:
		return FnFunc3(func(a, b, c any) any { return f(a, b, c) })
	case func(any, int, any) any:
		return FnFunc3(func(a, b, c any) any { return f(a, MustAsInt(b), c) })
	case func(any, any, any):
		return FnFunc3(func(a, b, c any) any { f(a, b, c); return nil })

	// --- 4 args ---
	case func(any, any, any, any) any:
		return FnFunc4(func(a, b, c, d any) any { return f(a, b, c, d) })
	}

	// Slow path: reflect.Value.Call with coercion for signatures not
	// covered by the type-switch above.
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
