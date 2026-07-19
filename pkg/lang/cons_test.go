package lang

import "testing"

func TestConsReduce(t *testing.T) {
	cons := NewCons(int64(1), NewList(int64(2), int64(3))).(*Cons)
	add := FnFunc2(func(a, b any) any { return Add(a, b) })

	if got := cons.Reduce(add); got != int64(6) {
		t.Fatalf("Reduce = %v, want 6", got)
	}
	if got := cons.ReduceInit(add, int64(4)); got != int64(10) {
		t.Fatalf("ReduceInit = %v, want 10", got)
	}
}
