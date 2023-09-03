package lang

import (
	"fmt"
	"testing"
)

func TestEquiv(t *testing.T) {
	equivs := [][]any{
		{nil, nil},
		{true, true},
		{false, false},
		{1, 1},
		{1.0, 1.0},
		{"a", "a"},
		{NewVector(), emptyList},
		{NewVector(1, 2, 3), NewList(1, 2, 3)},
		{NewPersistentHashMap(), emptyMap},
		{NewPersistentHashMap(1, 2, 3, 4), NewMap(1, 2, 3, 4), NewMap(3, 4, 1, 2)},
		{NewMap(1, 2).Seq(), NewVector(NewList(1, 2)), NewList(NewVector(1, 2))},
	}

	for _, els := range equivs {
		els := els
		t.Run(fmt.Sprintf("%v", els), func(t *testing.T) {
			t.Parallel()
			for i := range els {
				for j := range els {
					if !Equiv(els[i], els[j]) {
						t.Errorf("expected %v to equiv %v", els[i], els[j])
					}

					hasheqI := HashEq(els[i])
					hasheqJ := HashEq(els[j])
					// check hashes are equal
					if hasheqI != hasheqJ {
						t.Errorf("%v != %v, expected %v to hasheq to %v", hasheqI, hasheqJ, els[i], els[j])
					}
				}
			}
		})
	}
}
