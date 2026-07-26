package lang

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sync"
	"unicode"
)

type fomKey struct {
	ptr  uintptr
	name string
}

var fomCache sync.Map // fomKey -> interface{}

// fieldOrMethodResolver lets runtime-owned values provide a pre-resolved
// method table. It keeps hot host interop off the reflection and sync.Map
// paths while leaving the general Go interop behavior unchanged.
type fieldOrMethodResolver interface {
	fieldOrMethod(name string) (interface{}, bool)
}

// FieldOrMethodResolver lets packages built on lang provide direct method
// dispatch for runtime-owned values without manufacturing bound methods
// through reflect.
type FieldOrMethodResolver interface {
	ResolveFieldOrMethod(name string) (interface{}, bool)
}

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
	if resolver, ok := v.(FieldOrMethodResolver); ok {
		if result, found := resolver.ResolveFieldOrMethod(name); found {
			return result, true
		}
	}
	if resolver, ok := v.(fieldOrMethodResolver); ok {
		if result, found := resolver.fieldOrMethod(name); found {
			return result, true
		}
	}

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

	if result, ok := directProtocolMethod(v, name); ok {
		if canCache {
			fomCache.Store(key, result)
		}
		return result, true
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

func directProtocolMethod(v interface{}, name string) (IFn, bool) {
	switch name {
	case "Conj":
		if transient, ok := v.(Conjer); ok {
			return FnFunc1(func(value any) any {
				return transient.Conj(value)
			}), true
		}
	case "Persistent":
		if transient, ok := v.(ITransientCollection); ok {
			return FnFunc0(func() any {
				return transient.Persistent()
			}), true
		}
	case "AsTransient":
		if editable, ok := v.(IEditableCollection); ok {
			return FnFunc0(func() any {
				return editable.AsTransient()
			}), true
		}
	}
	if re, ok := v.(*regexp.Regexp); ok {
		switch name {
		case "MatchString":
			return FnFunc1(func(value any) any {
				return re.MatchString(value.(string))
			}), true
		case "FindStringSubmatch":
			return FnFunc1(func(value any) any {
				return re.FindStringSubmatch(value.(string))
			}), true
		case "FindStringSubmatchIndex":
			return FnFunc1(func(value any) any {
				return re.FindStringSubmatchIndex(value.(string))
			}), true
		case "ReplaceAllString":
			return FnFunc2(func(value, replacement any) any {
				return re.ReplaceAllString(
					value.(string),
					replacement.(string),
				)
			}), true
		case "Split":
			return FnFunc2(func(value, count any) any {
				return re.Split(value.(string), MustAsInt(count))
			}), true
		case "NumSubexp":
			return FnFunc0(func() any { return re.NumSubexp() }), true
		case "String":
			return FnFunc0(func() any { return re.String() }), true
		}
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
	case func() IPersistentCollection:
		return FnFunc0(func() any { return f() })
	case func() ITransientCollection:
		return FnFunc0(func() any { return f() })
	case func() IPersistentMap:
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
	case func(any) Conjer:
		return FnFunc1(func(a any) any { return f(a) })
	case func(any):
		return FnFunc1(func(a any) any { f(a); return nil })

	// --- 1 arg, typed param ---
	case func(string) string:
		return FnFunc1(func(a any) any { return f(a.(string)) })
	case func(string) *regexp.Regexp:
		return FnFunc1(func(a any) any { return f(a.(string)) })
	case func(string):
		return FnFunc1(func(a any) any { f(a.(string)); return nil })
	case func(IPersistentMap) any:
		return FnFunc1(func(a any) any {
			if a == nil {
				return f(nil)
			}
			return f(a.(IPersistentMap))
		})

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
	case func(int, int) any:
		return FnFunc2(func(a, b any) any { return f(MustAsInt(a), MustAsInt(b)) })
	case func(IFn, any) any:
		return FnFunc2(func(a, b any) any { return f(a.(IFn), b) })

	// --- 2 args, mixed typed ---
	case func(any, int) any:
		return FnFunc2(func(a, b any) any { return f(a, MustAsInt(b)) })
	case func(string, string) bool:
		return FnFunc2(func(a, b any) any {
			return f(a.(string), b.(string))
		})
	case func(Conser, any) Conser:
		return FnFunc2(func(a, b any) any { return f(asConser(a), b) })
	case func(*regexp.Regexp, string) *RegexpMatcher:
		return FnFunc2(func(a, b any) any {
			return f(a.(*regexp.Regexp), b.(string))
		})
	case func([]string, string) string:
		return FnFunc2(func(a, b any) any {
			return f(asStringSlice(a), b.(string))
		})
	case func(io.Writer, any) io.Writer:
		return FnFunc2(func(a, b any) any { return f(a.(io.Writer), b) })
	case func(string, int) Char:
		return FnFunc2(func(a, b any) any {
			return f(a.(string), MustAsInt(b))
		})

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

	// --- variadic args ---
	case func(any, any, ...any) any:
		return FnFunc(func(args ...any) any {
			if len(args) < 2 {
				panic(NewIllegalArgumentError(fmt.Sprintf(
					"wrong number of arguments: expected at least 2, got %d",
					len(args),
				)))
			}
			return f(args[0], args[1], args[2:]...)
		})
	case func(string, ...any) string:
		return FnFunc(func(args ...any) any {
			if len(args) < 1 {
				panic(NewIllegalArgumentError(
					"wrong number of arguments: expected at least 1, got 0",
				))
			}
			return f(args[0].(string), args[1:]...)
		})
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

// MustHostCast performs the assignable interface conversion used by generated
// direct host calls. A nil value becomes the zero value so nilable host
// parameters retain the same behavior as reflective invocation.
func MustHostCast[T any](value any) T {
	if value == nil {
		var zero T
		return zero
	}
	result, ok := value.(T)
	if !ok {
		panic(fmt.Errorf("cannot assign %T to host parameter", value))
	}
	return result
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
