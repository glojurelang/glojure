package lang

import (
	"reflect"
	"testing"
)

type applyToRecorder struct {
	args ISeq
}

func (*applyToRecorder) Invoke(...any) any {
	panic("ApplySeq unexpectedly called Invoke")
}

func (r *applyToRecorder) ApplyTo(args ISeq) any {
	r.args = args
	return int64(42)
}

func TestApplySeqUsesIFnApplyTo(t *testing.T) {
	args := NewLongRange(0, 10, 1)
	fn := &applyToRecorder{}

	if got := ApplySeq(fn, args); got != int64(42) {
		t.Fatalf("ApplySeq result = %v, want 42", got)
	}
	if fn.args != args {
		t.Fatal("ApplySeq did not pass the original sequence to ApplyTo")
	}
}

func TestApplySeqRetainsGoFunctionFallback(t *testing.T) {
	fn := func(a, b any) any {
		return Add(a, b)
	}
	args := NewList(int64(20), int64(22))

	if got := ApplySeq(fn, args); got != int64(42) {
		t.Fatalf("ApplySeq Go function result = %v, want 42", got)
	}
}

func TestApply2PreservesNilForTypedInterfaceParameter(t *testing.T) {
	if got := Apply2(Conj, nil, int64(3)); !Equals(got, NewList(int64(3))) {
		t.Fatalf("Apply2(Conj, nil, 3) = %v, want (3)", got)
	}
}

func TestApply5UsesFixedArityFunction(t *testing.T) {
	fn := FnFunc5(func(a, b, c, d, e any) any {
		return []any{a, b, c, d, e}
	})
	if got := Apply5(fn, 1, 2, 3, 4, 5).([]any); !reflect.DeepEqual(
		got,
		[]any{1, 2, 3, 4, 5},
	) {
		t.Fatalf("Apply5 result = %v", got)
	}
}

func TestApply9DoesNotAllocateForGeneratedFunction(t *testing.T) {
	fn := FnFunc9(func(a0, a1, a2, a3, a4, a5, a6, a7, a8 any) any {
		return a0.(int) + a1.(int) + a2.(int) + a3.(int) + a4.(int) +
			a5.(int) + a6.(int) + a7.(int) + a8.(int)
	})
	if got := Apply9(fn, 1, 2, 3, 4, 5, 6, 7, 8, 9); got != 45 {
		t.Fatalf("Apply9 result = %v, want 45", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		if Apply9(fn, 1, 2, 3, 4, 5, 6, 7, 8, 9) != 45 {
			panic("unexpected result")
		}
	}); got != 0 {
		t.Fatalf("Apply9 allocated %v objects per call, want 0", got)
	}
}
