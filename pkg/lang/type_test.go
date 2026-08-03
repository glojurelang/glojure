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

func TestHasTypeRecordDescriptor(t *testing.T) {
	recordType := InternRecordType("test", "TypedRecord", "value")
	if !HasType(recordType, NewRecord(recordType, 1)) {
		t.Fatal("record descriptor did not recognize its record")
	}
	otherType := InternRecordType("test", "OtherRecord", "value")
	if HasType(recordType, NewRecord(otherType, 1)) {
		t.Fatal("record descriptor recognized a different record")
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

func TestHasTypeAcceptsClassAliases(t *testing.T) {
	class := NewClassWithTypes(
		"example.Integer",
		reflect.TypeOf(int(0)),
		reflect.TypeOf(int32(0)),
	)
	if !HasType(class, int(1)) || !HasType(class, int32(1)) {
		t.Fatal("Class wrapper did not match all accepted host types")
	}
	if HasType(class, int64(1)) {
		t.Fatal("Class wrapper matched an unregistered host type")
	}
}
