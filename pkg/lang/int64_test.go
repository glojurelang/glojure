package lang

import (
	"math"
	"testing"
)

func TestCheckedInt64Arithmetic(t *testing.T) {
	if got := CheckedAddInt64(40, 2); got != 42 {
		t.Fatalf("CheckedAddInt64(40, 2) = %d", got)
	}
	if got := CheckedSubInt64(40, 2); got != 38 {
		t.Fatalf("CheckedSubInt64(40, 2) = %d", got)
	}
	if got := CheckedMultiplyInt64(6, 7); got != 42 {
		t.Fatalf("CheckedMultiplyInt64(6, 7) = %d", got)
	}
	if got := CheckedMultiplyInt64(-6, 7); got != -42 {
		t.Fatalf("CheckedMultiplyInt64(-6, 7) = %d", got)
	}
	if got := CheckedMultiplyInt64(math.MinInt64, 1); got != math.MinInt64 {
		t.Fatalf("CheckedMultiplyInt64(MinInt64, 1) = %d", got)
	}
	if got := CheckedMultiplyInt64(math.MinInt32, math.MinInt32); got != 1<<62 {
		t.Fatalf("CheckedMultiplyInt64(MinInt32, MinInt32) = %d", got)
	}
	if got := CheckedMultiplyInt64(math.MaxInt32, math.MaxInt32); got !=
		int64(math.MaxInt32)*int64(math.MaxInt32) {
		t.Fatalf("CheckedMultiplyInt64(MaxInt32, MaxInt32) = %d", got)
	}
	if got := CheckedNegateInt64(42); got != -42 {
		t.Fatalf("CheckedNegateInt64(42) = %d", got)
	}

	assertArithmeticPanic(t, func() { CheckedAddInt64(math.MaxInt64, 1) })
	assertArithmeticPanic(t, func() { CheckedSubInt64(math.MinInt64, 1) })
	assertArithmeticPanic(t, func() { CheckedMultiplyInt64(math.MaxInt64, 2) })
	assertArithmeticPanic(t, func() { CheckedMultiplyInt64(math.MinInt64, -1) })
	assertArithmeticPanic(t, func() { CheckedNegateInt64(math.MinInt64) })
}

func TestEqualsInt64MatchesDynamicNumericEquality(t *testing.T) {
	for _, test := range []struct {
		value any
		want  bool
	}{
		{value: int64(42), want: true},
		{value: int(42), want: true},
		{value: uint64(42), want: true},
		{value: int64(41), want: false},
		{value: "42", want: false},
		{value: nil, want: false},
	} {
		if got := EqualsInt64(42, test.value); got != test.want {
			t.Errorf(
				"EqualsInt64(42, %#v) = %v, want %v",
				test.value,
				got,
				test.want,
			)
		}
	}
	if EqualsInt64(-1, uint64(math.MaxUint64)) {
		t.Fatal("negative int64 equaled a large unsigned integer")
	}
}

func assertArithmeticPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected arithmetic panic")
		} else if _, ok := recovered.(*ArithmeticError); !ok {
			t.Fatalf("panic type = %T, want *ArithmeticError", recovered)
		}
	}()
	fn()
}
