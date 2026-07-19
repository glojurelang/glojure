package lang

import "math/bits"

// CheckedAddInt64 implements Clojure's overflowing fixed-width integer add
// without converting through interface values.
func CheckedAddInt64(a, b int64) int64 {
	result := a + b
	if (a^result)&(b^result) < 0 {
		panic(NewArithmeticError("integer overflow"))
	}
	return result
}

// CheckedSubInt64 implements Clojure's overflowing fixed-width integer
// subtraction without converting through interface values.
func CheckedSubInt64(a, b int64) int64 {
	result := a - b
	if (a^b)&(a^result) < 0 {
		panic(NewArithmeticError("integer overflow"))
	}
	return result
}

// CheckedMultiplyInt64 implements Clojure's overflowing fixed-width integer
// multiplication without converting through interface values.
func CheckedMultiplyInt64(a, b int64) int64 {
	unsignedA, unsignedB := uint64(a), uint64(b)
	high, low := bits.Mul64(unsignedA, unsignedB)
	high -= uint64(a>>63) & unsignedB
	high -= uint64(b>>63) & unsignedA
	result := int64(low)
	if int64(high) != result>>63 {
		panic(NewArithmeticError("integer overflow"))
	}
	return result
}

// CheckedNegateInt64 implements Clojure's overflowing fixed-width integer
// negation without converting through an interface value.
func CheckedNegateInt64(value int64) int64 {
	if value == -1<<63 {
		panic(NewArithmeticError("integer overflow"))
	}
	return -value
}
