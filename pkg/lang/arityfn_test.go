package lang

import "testing"

func TestArityFnUsesFixedArityMethods(t *testing.T) {
	fn := NewArityFn(
		FnFunc0(func() any { return "zero" }),
		nil,
		FnFunc2(func(a, b any) any { return a.(int) + b.(int) }),
		nil,
		nil,
		FnFunc(func(args ...any) any { return len(args) }),
		3,
	)

	if got := Apply0(fn); got != "zero" {
		t.Fatalf("Apply0 = %v, want zero", got)
	}
	if got := Apply2(fn, 2, 3); got != 5 {
		t.Fatalf("Apply2 = %v, want 5", got)
	}
	if got := Apply3(fn, 1, 2, 3); got != 3 {
		t.Fatalf("variadic Apply3 = %v, want 3", got)
	}
}

func TestArityFnApplyToPassesSequenceToVariadicMethod(t *testing.T) {
	args := NewLongRange(0, 10_000_000, 1)
	var applied ISeq
	variadic := NewVariadicFn(
		1,
		func(fixed []any, rest ISeq) any {
			applied = rest
			return fixed[0]
		},
	)
	fn := NewArityFn(nil, nil, nil, nil, nil, variadic, 1)

	if got := fn.ApplyTo(args); got != int64(0) {
		t.Fatalf("ApplyTo result = %v, want 0", got)
	}
	if applied.First() != int64(1) {
		t.Fatal("ApplyTo did not pass the unrealized sequence tail to the variadic method")
	}
}

func TestArityFnApplyToPrefersExactFixedArity(t *testing.T) {
	variadic := NewVariadicFn(
		2,
		func(fixed []any, rest ISeq) any { return "variadic" },
	)
	fn := NewArityFn(
		nil,
		nil,
		FnFunc2(func(a, b any) any { return Add(a, b) }),
		nil,
		nil,
		variadic,
		2,
	)

	if got := fn.ApplyTo(NewList(int64(20), int64(22))); got != int64(42) {
		t.Fatalf("ApplyTo result = %v, want 42", got)
	}
}

func TestArityFnMethodsDispatchesHigherFixedArity(t *testing.T) {
	variadic := NewVariadicFn(
		6,
		func(fixed []any, rest ISeq) any { return "variadic" },
	)
	fn := NewArityFnMethods(
		map[int]IFn{
			6: FnFunc(func(args ...any) any { return "fixed" }),
		},
		variadic,
		6,
	)

	if got := fn.ApplyTo(NewList(0, 1, 2, 3, 4, 5)); got != "fixed" {
		t.Fatalf("six-argument ApplyTo = %v, want fixed", got)
	}
	if got := fn.ApplyTo(NewList(0, 1, 2, 3, 4, 5, 6)); got != "variadic" {
		t.Fatalf("seven-argument ApplyTo = %v, want variadic", got)
	}
}
