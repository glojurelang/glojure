package lang

import (
	"reflect"
	"testing"
)

func TestSeqToTypedArrayInfersComponentType(t *testing.T) {
	got := SeqToTypedArray(NewList(int64(1), int64(2), int64(3)))
	want := []int64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SeqToTypedArray = %#v, want %#v", got, want)
	}
}

func TestSeqToTypedArrayUsesExplicitComponentType(t *testing.T) {
	got := SeqToTypedArray(BuiltinTypes["float32"], NewList(1, 2, 3))
	want := []float32{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SeqToTypedArray = %#v, want %#v", got, want)
	}
}

func TestSeqToTypedArrayEmptySequenceUsesAny(t *testing.T) {
	got := SeqToTypedArray(nil)
	if typ := reflect.TypeOf(got); typ != reflect.TypeOf([]any{}) {
		t.Fatalf("empty array type = %v, want []any", typ)
	}
}

func TestSliceSetAcceptsNilForReferenceArrays(t *testing.T) {
	values := []any{"old"}
	SliceSet(values, 0, nil)
	if values[0] != nil {
		t.Fatalf("value = %v, want nil", values[0])
	}
}
