package lang

import (
	"testing"
)

type testReceiver struct {
	Value int
}

func (r *testReceiver) Double() int {
	return r.Value * 2
}

func (r *testReceiver) Add(n int) int {
	return r.Value + n
}

func TestFieldOrMethodCacheReturnsSameValue(t *testing.T) {
	r := &testReceiver{Value: 5}
	v1, ok1 := FieldOrMethod(r, "double")
	v2, ok2 := FieldOrMethod(r, "double")
	if !ok1 || !ok2 {
		t.Fatal("FieldOrMethod returned false")
	}
	// Results should both implement IFn.
	fn1, ok := v1.(IFn)
	if !ok {
		t.Fatalf("FieldOrMethod did not return IFn, got %T", v1)
	}
	fn2 := v2.(IFn)
	// Can't compare funcs directly, but verify both produce correct results.
	r1 := fn1.Invoke()
	r2 := fn2.Invoke()
	if r1 != 10 || r2 != 10 {
		t.Errorf("Double() = %v, %v; want 10, 10", r1, r2)
	}
}

func TestFieldOrMethodCachedFnFuncCorrectResults(t *testing.T) {
	r := &testReceiver{Value: 7}
	v, ok := FieldOrMethod(r, "add")
	if !ok {
		t.Fatal("FieldOrMethod returned false for Add")
	}
	fn := v.(IFn)
	result := fn.Invoke(3)
	if result != 10 {
		t.Errorf("Add(3) = %v, want 10", result)
	}
}

func TestFieldOrMethodDifferentReceiversCached(t *testing.T) {
	r1 := &testReceiver{Value: 1}
	r2 := &testReceiver{Value: 2}
	v1, _ := FieldOrMethod(r1, "double")
	v2, _ := FieldOrMethod(r2, "double")
	fn1 := v1.(IFn)
	fn2 := v2.(IFn)
	// Both should work — different receivers bind different methods.
	res1 := fn1.Invoke()
	res2 := fn2.Invoke()
	if res1 != 2 {
		t.Errorf("r1.Double() = %v, want 2", res1)
	}
	if res2 != 4 {
		t.Errorf("r2.Double() = %v, want 4", res2)
	}
}

func TestFieldOrMethodReturnsField(t *testing.T) {
	r := &testReceiver{Value: 42}
	v, ok := FieldOrMethod(r, "value")
	if !ok {
		t.Fatal("FieldOrMethod returned false for Value field")
	}
	// Fields are not wrapped as IFn.
	if _, isFn := v.(IFn); isFn {
		t.Error("Field should not be wrapped as IFn")
	}
	if v != 42 {
		t.Errorf("Value = %v, want 42", v)
	}
}

func TestFieldOrMethodNotFound(t *testing.T) {
	r := &testReceiver{Value: 1}
	_, ok := FieldOrMethod(r, "nonexistent")
	if ok {
		t.Error("FieldOrMethod returned true for nonexistent field/method")
	}
}

// Test wrapGoFunc type-switch covers common signatures.
func TestWrapGoFuncDirectCall(t *testing.T) {
	tests := []struct {
		name string
		fn   interface{}
		args []any
		want any
	}{
		{"func()any", func() any { return 42 }, nil, 42},
		{"func()int", func() int { return 7 }, nil, 7},
		{"func()bool", func() bool { return true }, nil, true},
		{"func(any)any", func(x any) any { return x }, []any{"hi"}, "hi"},
		{"func(any)bool", func(x any) bool { return x == 1 }, []any{1}, true},
		{"func(any)int", func(x any) int { return x.(int) * 2 }, []any{5}, 10},
		{"func(any,any)any", func(a, b any) any { return a.(int) + b.(int) }, []any{3, 4}, 7},
		{"func(any,any)bool", func(a, b any) bool { return a == b }, []any{1, 1}, true},
		{"func(any,any,any)any", func(a, b, c any) any { return a.(int) + b.(int) + c.(int) }, []any{1, 2, 3}, 6},
		{"func(string)string", func(s string) string { return s + "!" }, []any{"hi"}, "hi!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := wrapGoFunc(tt.fn)
			got := fn.Invoke(tt.args...)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapGoFuncVoidReturn(t *testing.T) {
	called := false
	fn := wrapGoFunc(func(x any) { called = true })
	result := fn.Invoke("test")
	if !called {
		t.Error("void function was not called")
	}
	if result != nil {
		t.Errorf("void function returned %v, want nil", result)
	}
}
