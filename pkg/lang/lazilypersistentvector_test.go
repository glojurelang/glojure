package lang

import "testing"

type vectorReductionSource struct{ called bool }

func (s *vectorReductionSource) ReduceInit(f IFn, init any) any {
	s.called = true
	for _, item := range []any{1, 2} {
		init = Apply2(f, init, item)
	}
	return init
}

func (s *vectorReductionSource) Seq() ISeq {
	panic("vector construction must honor IReduceInit")
}

func TestVectorConstructionUsesReductionContract(t *testing.T) {
	source := &vectorReductionSource{}
	result := CreateLazilyPersistentVector(source)
	if !source.called || result.Count() != 2 || result.Nth(1) != 2 {
		t.Fatal("vector construction ignored the reduction result")
	}
}

func TestVectorConstructionBoundariesAndSnapshots(t *testing.T) {
	for _, count := range []int{0, 1, 31, 32, 33, 1023, 1024, 1025, 2500} {
		values := make([]any, count)
		for i := range values {
			values[i] = i
		}
		mapped := NewMappedSeq(FnFunc1(func(value any) any { return value }), NewSliceSeq(values))
		result := CreateLazilyPersistentVector(mapped)
		owned := CreateOwningLazilyPersistentVector(values)
		if result.Count() != count || owned.Count() != count {
			t.Fatalf("construction at %d changed count", count)
		}
		for i := range values {
			if result.Nth(i) != i || owned.Nth(i) != i {
				t.Fatalf("construction at %d changed element %d", count, i)
			}
		}
		if count != 0 {
			values[0] = -1
			if result.Nth(0) != 0 || owned.Nth(0) != 0 {
				t.Fatal("construction retained a mutable input slice")
			}
			changed := result.AssocN(count/2, -2)
			if result.Nth(count/2) != count/2 || changed.Nth(count/2) != -2 {
				t.Fatal("derived vector changed a persistent snapshot")
			}
		}
	}
}

func TestVectorConstructionFromTypedHostArrays(t *testing.T) {
	for _, source := range []any{[]int{1, 2, 3}, [3]int{1, 2, 3}} {
		result := CreateOwningLazilyPersistentVector(source)
		if !Equals(result, NewVector(1, 2, 3)) {
			t.Fatalf("typed construction from %T = %v", source, result)
		}
	}
}
