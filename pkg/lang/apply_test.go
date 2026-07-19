package lang

import "testing"

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
