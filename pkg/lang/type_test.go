package lang

import "testing"

func TestIsInstance(t *testing.T) {
	vector := NewVector(int64(1))
	if !IsInstance[IPersistentVector](vector) {
		t.Fatal("vector did not satisfy IPersistentVector")
	}
	if IsInstance[IPersistentVector](NewMap()) {
		t.Fatal("map unexpectedly satisfied IPersistentVector")
	}
	if !IsInstance[string]("value") {
		t.Fatal("string did not satisfy its concrete type")
	}
	if IsInstance[string](nil) {
		t.Fatal("nil unexpectedly satisfied string")
	}
}
