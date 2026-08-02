package lang

import (
	"reflect"
	"testing"
)

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

func TestHasTypeAcceptsClass(t *testing.T) {
	class := NewClass(reflect.TypeOf(""), "java.lang.String")
	if !HasType(class, "value") {
		t.Fatal("Class wrapper did not match its host value")
	}
	if HasType(class, int64(1)) {
		t.Fatal("Class wrapper matched the wrong host value")
	}
}
