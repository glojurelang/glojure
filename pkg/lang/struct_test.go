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
	// Same FnFunc instance from cache.
	fn1 := v1.(FnFunc)
	fn2 := v2.(FnFunc)
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
	fn := v.(FnFunc)
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
	fn1 := v1.(FnFunc)
	fn2 := v2.(FnFunc)
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
	// Fields are not wrapped as FnFunc.
	if _, isFn := v.(FnFunc); isFn {
		t.Error("Field should not be wrapped as FnFunc")
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
