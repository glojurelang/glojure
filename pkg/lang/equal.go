package lang

import "reflect"

func Equiv(a, b any) bool {
	return Equals(a, b)
}

func Equals(a, b any) bool {
	// check functions first, because == panics on func comparison.
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)
	if aVal.Kind() == reflect.Func || bVal.Kind() == reflect.Func {
		if !(aVal.Kind() == reflect.Func && bVal.Kind() == reflect.Func) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}
	if aVal.Kind() == reflect.Map || bVal.Kind() == reflect.Map || aVal.Kind() == reflect.Slice || bVal.Kind() == reflect.Slice {
		if aVal.Kind() != bVal.Kind() {
			return false
		}
		return Equals(Seq(a), Seq(b))
	}

	if a == b {
		return true
	}

	aNil, bNil := IsNil(a), IsNil(b)

	if aNil && bNil {
		// both nil
		return true
	}
	if aNil || bNil {
		// one nil
		return false
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
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)

	// check if comparing functions, because == panics on func comparison.
	if aVal.Kind() == reflect.Func || bVal.Kind() == reflect.Func {
		if !(aVal.Kind() == reflect.Func && bVal.Kind() == reflect.Func) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}
	// slices
	if aVal.Kind() == reflect.Slice || bVal.Kind() == reflect.Slice {
		if !(aVal.Kind() == reflect.Slice && bVal.Kind() == reflect.Slice) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}
	// arrays
	if aVal.Kind() == reflect.Array || bVal.Kind() == reflect.Array {
		if !(aVal.Kind() == reflect.Array && bVal.Kind() == reflect.Array) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}
	// maps
	if aVal.Kind() == reflect.Map || bVal.Kind() == reflect.Map {
		if !(aVal.Kind() == reflect.Map && bVal.Kind() == reflect.Map) {
			return false
		}
		return aVal.Pointer() == bVal.Pointer()
	}

	return a == b
}

func pcEquiv(a, b any) bool {
	if a, ok := a.(IPersistentCollection); ok {
		return a.Equiv(b)
	}
	return b.(IPersistentCollection).Equiv(a)
}
