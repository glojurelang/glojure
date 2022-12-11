package value

type equaler interface {
	Equal(interface{}) bool
}

// Equal returns true if the two values are equal.
func Equal(a, b interface{}) bool {
	if a == b {
		return true
	}
	if _, ok := AsNumber(a); ok {
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
