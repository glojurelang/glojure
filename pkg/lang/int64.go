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
	// The product of two signed 32-bit values always fits in int64. This common
	// case avoids computing the high half solely to prove sign extension.
	if int64(int32(a)) == a && int64(int32(b)) == b {
		return a * b
	}
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

// ModInt64 implements clojure.core/mod for fixed-width integers. Go's
// remainder has the dividend's sign; Clojure's modulus has the divisor's
// sign and therefore needs one adjustment when their signs differ.
func ModInt64(num, div int64) int64 {
	if div == 0 {
		panic(NewArithmeticError("divide by zero"))
	}
	remainder := num % div
	if remainder == 0 || (num > 0) == (div > 0) {
		return remainder
	}
	return remainder + div
}
