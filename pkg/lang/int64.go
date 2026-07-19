package lang

// CheckedAddInt64 implements Clojure's overflowing fixed-width integer add
// without converting through interface values.
func CheckedAddInt64(a, b int64) int64 {
	result := a + b
	if (result > a) == (b > 0) {
		return result
	}
	panic(NewArithmeticError("integer overflow"))
}

// CheckedSubInt64 implements Clojure's overflowing fixed-width integer
// subtraction without converting through interface values.
func CheckedSubInt64(a, b int64) int64 {
	result := a - b
	if (result < a) == (b > 0) {
		return result
	}
	panic(NewArithmeticError("integer overflow"))
}

// CheckedMultiplyInt64 implements Clojure's overflowing fixed-width integer
// multiplication without converting through interface values.
func CheckedMultiplyInt64(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	result := a * b
	if (result < 0) == ((a < 0) != (b < 0)) && result/b == a {
		return result
	}
	panic(NewArithmeticError("integer overflow"))
}

// CheckedNegateInt64 implements Clojure's overflowing fixed-width integer
// negation without converting through an interface value.
func CheckedNegateInt64(value int64) int64 {
	if value == -1<<63 {
		panic(NewArithmeticError("integer overflow"))
	}
	return -value
}
