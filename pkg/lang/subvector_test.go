package lang

import "testing"

func TestSubVectorReduce(t *testing.T) {
	vector := NewVector(int64(1), int64(2), int64(3), int64(4))
	subvector := NewSubVector(nil, vector, 1, 4)
	add := FnFunc2(func(left, right any) any {
		return left.(int64) + right.(int64)
	})
	if got := subvector.ReduceInit(add, int64(0)); got != int64(9) {
		t.Fatalf("ReduceInit = %v, want 9", got)
	}
	if got := subvector.Reduce(add); got != int64(9) {
		t.Fatalf("Reduce = %v, want 9", got)
	}
}
