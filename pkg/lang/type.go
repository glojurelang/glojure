package lang

import (
	"bytes"
	"reflect"
	"regexp"
)

var (
	Throwable = reflect.TypeOf((interface{})(nil))

	// Exception types
	IllegalStateException = reflect.TypeOf(&IllegalStateError{})

	// TODO: convert use of 'matcher' in core.glj to fit go's
	// regexps. This supresses errors but doesn't actually work.
	Matcher = reflect.TypeOf(&regexp.Regexp{})

	// TODO: rework use of PrintWriter in core.glj
	PrintWriter = reflect.TypeOf(&bytes.Buffer{})
)

func HasType(target any, v interface{}) bool {
	if v == nil {
		return false
	}
	t, ok := ReflectType(target)
	if !ok {
		return false
	}
	vType := reflect.TypeOf(v)
	switch {
	case vType == t, vType.AssignableTo(t):
		return true
	default:
		return false
	}
}

// ReflectType unwraps both native Go reflect.Type values and JVM-style Class
// wrappers to the host type used by the runtime.
func ReflectType(value any) (reflect.Type, bool) {
	switch v := value.(type) {
	case *Class:
		if v == nil {
			return nil, false
		}
		return v.Type, true
	case reflect.Type:
		return v, true
	default:
		return nil, false
	}
}

// IsInstance performs the same assignability check as HasType when the target
// type is known at compile time. AOT code can use a Go type assertion instead
// of reflect.Type.AssignableTo.
func IsInstance[T any](v interface{}) bool {
	_, ok := v.(T)
	return ok
}

func TypeOf(v interface{}) reflect.Type {
	return reflect.TypeOf(v)
}
