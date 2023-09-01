package lang

import "reflect"

func Equiv(a, b any) bool {
	return Equal(a, b)
}

func Equals(a, b any) bool {
	return Equal(a, b)
}

// Equal returns true if the two values are equivalent.
// TODO: rename to Equiv.
func Equal(a, b any) bool {
	// check functions first, because == panics on func comparison.
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)
	if aVal.Kind() == reflect.Func || bVal.Kind() == reflect.Func {
		if !(aVal.Kind() == reflect.Func && bVal.Kind() == reflect.Func) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}

	if a == b || aVal.Kind() == reflect.Ptr && aVal.IsNil() && bVal.Kind() == reflect.Ptr && bVal.IsNil() {
		return true
	}
	if _, ok := AsNumber(a); ok {
		if _, ok := AsNumber(b); !ok {
			return false
		}
		return NumbersEqual(a, b)
	}
	if _, ok := a.(IPersistentCollection); ok {
		return pcEquiv(a, b)
	}
	if _, ok := b.(IPersistentCollection); ok {
		return pcEquiv(a, b)
	}

	if a, ok := a.(Equalser); ok {
		return a.Equals(b)
	}
	if b, ok := b.(Equalser); ok {
		return b.Equals(a)
	}

	if a, ok := a.(Equiver); ok {
		return a.Equiv(b)
	}
	if b, ok := b.(Equiver); ok {
		return b.Equiv(a)
	}

	// TODO: match all clojure equality rules

	return false
}

func Identical(a, b any) bool {
	return a == b
}

func pcEquiv(a, b any) bool {
	if a, ok := a.(IPersistentCollection); ok {
		return a.Equiv(b)
	}
	return b.(IPersistentCollection).Equiv(a)
}
