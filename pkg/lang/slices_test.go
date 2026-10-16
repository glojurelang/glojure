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

type tracedSliceSeq struct {
	ISeq
	steps *int
}

func (s tracedSliceSeq) Seq() ISeq { return s }

func (s tracedSliceSeq) Next() ISeq {
	*s.steps++
	next := s.ISeq.Next()
	if next == nil {
		return nil
	}
	return tracedSliceSeq{ISeq: next, steps: s.steps}
}

func TestToSliceTraversesUncountedSequenceOnce(t *testing.T) {
	steps := 0
	value := tracedSliceSeq{
		ISeq:  NewSliceSeq([]any{int64(1), int64(2), int64(3)}),
		steps: &steps,
	}
	want := []any{int64(1), int64(2), int64(3)}
	if got := ToSlice(value); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToSlice = %v, want %v", got, want)
	}
	if steps != len(want) {
		t.Fatalf("Next called %d times, want %d", steps, len(want))
	}
	empty := ToSlice(NewLazySeq(func() any { return nil }))
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty sequence produced %#v, want non-nil empty slice", empty)
	}
}
