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
