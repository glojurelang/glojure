package arrays

import (
	"reflect"
	"testing"
)

func TestNewInstance(t *testing.T) {
	bytes := NewInstance(reflect.TypeOf(int8(0)), int64(3))
	if got := reflect.TypeOf(bytes); got != reflect.TypeOf([]int8{}) {
		t.Fatalf("byte array type = %v, want []int8", got)
	}
	matrix := NewInstance(reflect.TypeOf(""), []int32{2, 3})
	value := reflect.ValueOf(matrix)
	if value.Len() != 2 || value.Index(0).Len() != 3 {
		t.Fatalf("matrix dimensions = %dx%d, want 2x3", value.Len(), value.Index(0).Len())
	}
}

func TestEqualsByteRepresentations(t *testing.T) {
	if !Equals([]byte{0, 255}, []int8{0, -1}) {
		t.Fatal("signed and unsigned byte storage should compare equally")
	}
	if Equals([]byte{1}, []byte{2}) {
		t.Fatal("different arrays compared equal")
	}
}
