package lang

import (
	"testing"
)

// TestFnFunc0 verifies FnFunc0 implements IFn correctly.
func TestFnFunc0(t *testing.T) {
	called := false
	f := FnFunc0(func() any {
		called = true
		return 42
	})

	// Invoke with correct arity
	result := f.Invoke()
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
	if !called {
		t.Error("function was not called")
	}

	// Invoke with wrong arity panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for wrong arity")
			}
		}()
		f.Invoke(1)
	}()
}

// TestFnFunc1 verifies FnFunc1 implements IFn correctly.
func TestFnFunc1(t *testing.T) {
	f := FnFunc1(func(a any) any {
		return a.(int) * 2
	})

	result := f.Invoke(21)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}

	// Wrong arity panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for wrong arity")
			}
		}()
		f.Invoke()
	}()
}

// TestFnFunc2 verifies FnFunc2 implements IFn correctly.
func TestFnFunc2(t *testing.T) {
	f := FnFunc2(func(a, b any) any {
		return a.(int) + b.(int)
	})

	result := f.Invoke(20, 22)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}

	// Wrong arity panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for wrong arity")
			}
		}()
		f.Invoke(1)
	}()
}

// TestFnFunc3 verifies FnFunc3 implements IFn correctly.
func TestFnFunc3(t *testing.T) {
	f := FnFunc3(func(a, b, c any) any {
		return a.(int) + b.(int) + c.(int)
	})

	result := f.Invoke(10, 20, 12)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

// TestFnFunc4 verifies FnFunc4 implements IFn correctly.
func TestFnFunc4(t *testing.T) {
	f := FnFunc4(func(a, b, c, d any) any {
		return a.(int) + b.(int) + c.(int) + d.(int)
	})

	result := f.Invoke(10, 10, 11, 11)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

// TestApply0 verifies Apply0 dispatches correctly.
func TestApply0(t *testing.T) {
	// FnFunc0 fast path — zero allocation
	f0 := FnFunc0(func() any { return "zero" })
	if got := Apply0(f0); got != "zero" {
		t.Errorf("Apply0(FnFunc0): expected %q, got %v", "zero", got)
	}

	// FnFunc fallback
	ff := FnFunc(func(args ...any) any { return "fnfunc" })
	if got := Apply0(ff); got != "fnfunc" {
		t.Errorf("Apply0(FnFunc): expected %q, got %v", "fnfunc", got)
	}

	// nil panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Apply0(nil) should panic")
			}
		}()
		Apply0(nil)
	}()
}

// TestApply1 verifies Apply1 dispatches correctly.
func TestApply1(t *testing.T) {
	f1 := FnFunc1(func(a any) any { return a.(int) + 1 })
	if got := Apply1(f1, 41); got != 42 {
		t.Errorf("Apply1(FnFunc1): expected 42, got %v", got)
	}

	ff := FnFunc(func(args ...any) any { return args[0].(int) * 2 })
	if got := Apply1(ff, 21); got != 42 {
		t.Errorf("Apply1(FnFunc): expected 42, got %v", got)
	}
}

// TestApply2 verifies Apply2 dispatches correctly.
func TestApply2(t *testing.T) {
	f2 := FnFunc2(func(a, b any) any { return a.(int) + b.(int) })
	if got := Apply2(f2, 20, 22); got != 42 {
		t.Errorf("Apply2(FnFunc2): expected 42, got %v", got)
	}

	ff := FnFunc(func(args ...any) any { return args[0].(int) + args[1].(int) })
	if got := Apply2(ff, 10, 32); got != 42 {
		t.Errorf("Apply2(FnFunc): expected 42, got %v", got)
	}
}

// TestApply3 verifies Apply3 dispatches correctly.
func TestApply3(t *testing.T) {
	f3 := FnFunc3(func(a, b, c any) any { return a.(int) + b.(int) + c.(int) })
	if got := Apply3(f3, 10, 20, 12); got != 42 {
		t.Errorf("Apply3(FnFunc3): expected 42, got %v", got)
	}
}

// TestApply4 verifies Apply4 dispatches correctly.
func TestApply4(t *testing.T) {
	f4 := FnFunc4(func(a, b, c, d any) any {
		return a.(int) + b.(int) + c.(int) + d.(int)
	})
	if got := Apply4(f4, 10, 10, 11, 11); got != 42 {
		t.Errorf("Apply4(FnFunc4): expected 42, got %v", got)
	}
}

// TestApplyNFallbackToIFn verifies ApplyN falls back to IFn for non-FnFuncN types.
func TestApplyNFallbackToIFn(t *testing.T) {
	// Use a keyword (which implements IFn) as a function
	// Keyword.Invoke with wrong args will panic, so use a custom IFn
	type testIFn struct{}
	// We'll use FnFunc as our IFn test subject since it implements IFn
	_ = testIFn{}

	// Test that Apply2 works with a generic IFn (not FnFunc2)
	generic := FnFunc(func(args ...any) any {
		sum := 0
		for _, a := range args {
			sum += a.(int)
		}
		return sum
	})
	if got := Apply2(generic, 20, 22); got != 42 {
		t.Errorf("Apply2(generic IFn): expected 42, got %v", got)
	}
}

// TestNewFnFuncN verifies the constructor helpers.
func TestNewFnFuncN(t *testing.T) {
	f0 := NewFnFunc0(func() any { return 0 })
	f1 := NewFnFunc1(func(a any) any { return a })
	f2 := NewFnFunc2(func(a, b any) any { return a })
	f3 := NewFnFunc3(func(a, b, c any) any { return a })
	f4 := NewFnFunc4(func(a, b, c, d any) any { return a })

	fns := []struct {
		name string
		fn   any
	}{
		{"FnFunc0", f0}, {"FnFunc1", f1}, {"FnFunc2", f2},
		{"FnFunc3", f3}, {"FnFunc4", f4},
	}
	for _, tc := range fns {
		if _, ok := tc.fn.(IFn); !ok {
			t.Errorf("%s does not implement IFn", tc.name)
		}
	}
}
