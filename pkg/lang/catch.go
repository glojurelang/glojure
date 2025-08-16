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
	if IsNil(expect) {
		return false
	}

	// Special case: the symbol "any" catches everything (for go/any)
	if sym, ok := expect.(*Symbol); ok && sym.Name() == "any" {
		return true
	}

	// If expect is an error type, check if r is an instance of it
	if rErr, ok := r.(error); ok {
		if expectTyp, ok := expect.(reflect.Type); ok && expectTyp.Implements(errorType) {
			expectVal := reflect.New(expectTyp).Elem().Interface().(error)
			if errors.Is(rErr, expectVal) {
				return true
			}
		}
	}

	// General type check
	if expectTyp, ok := expect.(reflect.Type); ok {
		return reflect.TypeOf(r).AssignableTo(expectTyp)
	}

	// For interface{} type (go/any), catch everything
	if expectTyp, ok := expect.(reflect.Type); ok && expectTyp.Kind() == reflect.Interface && expectTyp.NumMethod() == 0 {
		return true
	}

	return false
}