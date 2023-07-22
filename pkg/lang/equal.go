package lang

import "reflect"

type equaler interface {
	Equal(interface{}) bool
}

// Equal returns true if the two values are equal.
func Equal(a, b interface{}) bool {
	// check functions first, because == panics on func comparison.
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)
	if aVal.Kind() == reflect.Func || bVal.Kind() == reflect.Func {
		if !(aVal.Kind() == reflect.Func && bVal.Kind() == reflect.Func) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}

	if a == b || IsNil(a) && IsNil(b) {
		return true
	}
	if _, ok := AsNumber(a); ok {
		if _, ok := AsNumber(b); !ok {
			return false
		}
		return NumbersEqual(a, b)
	}

	if a, ok := a.(equaler); ok {
		return a.Equal(b)
	}
	if b, ok := b.(equaler); ok {
		return b.Equal(a)
	}

	// TODO: match all clojure equality rules

	return false
}

func Identical(a, b interface{}) bool {
	return a == b
}
