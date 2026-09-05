package lang

import (
	"reflect"
	"testing"
)

func TestBuiltinSliceHasFixedArities(t *testing.T) {
	b := Builtins["slice"]
	fn, ok := b.(ArityFn)
	if !ok {
		t.Fatalf("Builtins[\"slice\"] is %T, want ArityFn", b)
	}
	if got := fn.Invoke2("hello", int64(2)); got != "llo" {
		t.Errorf("slice(\"hello\", 2) = %q, want \"llo\"", got)
	}
	if got := fn.Invoke3("hello", nil, int64(2)); got != "he" {
		t.Errorf("slice(\"hello\", nil, 2) = %q, want \"he\"", got)
	}
	got := fn.Invoke3([]int{1, 2, 3}, int64(1), int64(3))
	if !reflect.DeepEqual(got, []int{2, 3}) {
		t.Errorf("slice([1 2 3], 1, 3) = %v, want [2 3]", got)
	}
}

func TestBuiltinAppendIsFnFunc(t *testing.T) {
	b := Builtins["append"]
	if _, ok := b.(FnFunc); !ok {
		t.Errorf("Builtins[\"append\"] is %T, want FnFunc", b)
	}
}

func TestBuiltinTypeEntriesAreReflectType(t *testing.T) {
	for _, name := range []string{"int", "string", "bool", "float64"} {
		b := Builtins[name]
		if _, ok := b.(reflect.Type); !ok {
			t.Errorf("Builtins[%q] is %T, want reflect.Type", name, b)
		}
	}
}

func TestBuiltinSliceWorksCorrectly(t *testing.T) {
	fn := Builtins["slice"].(IFn)
	result := fn.Invoke("hello", 1, 3)
	if result != "el" {
		t.Errorf("slice(\"hello\", 1, 3) = %q, want \"el\"", result)
	}
}

func TestBuiltinLenWorksCorrectly(t *testing.T) {
	fn := Builtins["len"].(IFn)
	result := fn.Invoke([]int{1, 2, 3})
	if result != 3 {
		t.Errorf("len([1,2,3]) = %v, want 3", result)
	}
}

func TestBuiltinDeleteWorksCorrectly(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	fn := Builtins["delete"].(IFn)
	result := fn.Invoke(m, "a")
	if result != nil {
		t.Errorf("delete() returned %v, want nil", result)
	}
	if _, ok := m["a"]; ok {
		t.Errorf("key \"a\" still present after delete")
	}
}

func TestBuiltinApplyUsesIFnPath(t *testing.T) {
	// Verify that Apply routes through IFn for builtins.
	fn := Builtins["len"]
	if _, ok := fn.(IFn); !ok {
		t.Fatal("Builtins[\"len\"] does not implement IFn")
	}
	result := Apply(fn, []interface{}{[]int{10, 20}})
	if result != 2 {
		t.Errorf("Apply(len, [[10,20]]) = %v, want 2", result)
	}
}
