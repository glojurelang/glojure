package lang

import (
	"errors"
	"reflect"
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

// CatchMatches checks if a recovered panic value matches an expected catch type.
// This implements the semantics of Clojure's try/catch matching.
func CatchMatches(r, expect any) bool {
	if IsNil(expect) || IsNil(r) {
		return false
	}

	expectType := expect.(reflect.Type)

	// if expect is an error type, check if r is an instance of it
	if rErr, ok := r.(error); ok {
		if expectType.Implements(errorType) {
			// if expectType is a pointer type, instantiate a new value of that type
			// and check if rErr is an instance of it
			if expectType.Kind() == reflect.Ptr {
				if errors.As(rErr, reflect.New(expectType).Interface()) {
					return true
				}
			}
			// if expectType is an interface type, check if rErr implements it
			if expectType.Kind() == reflect.Interface {
				if errors.As(rErr, reflect.New(expectType).Interface()) {
					return true
				}
			}
			// otherwise, create a new value of the expectType and check if
			// rErr is an instance of it
			expectVal := reflect.New(expectType).Interface()
			if errors.As(rErr, expectVal) {
				return true
			}
		}
	}

	return reflect.TypeOf(r).AssignableTo(expectType)
}
