package lang

import (
	"fmt"
	"testing"
)

type charEqualser struct{}

func (charEqualser) Equals(value interface{}) bool {
	_, ok := value.(Char)
	return ok
}

func TestEquiv(t *testing.T) {
	equivs := [][]any{
		{nil, nil},
		{true, true},
		{false, false},
		{1, 1},
		{1.0, 1.0},
		{"a", "a"},
		{NewChar('A'), BoxChar('A')},
		{NewVector(), emptyList},
		{NewVector(1, 2, 3), NewList(1, 2, 3)},
		{NewPersistentHashMap(), emptyMap},
		{NewPersistentHashMap(1, 2, 3, 4), NewMap(1, 2, 3, 4), NewMap(3, 4, 1, 2)},
		{NewMap(1, 2).Seq(), NewVector(NewList(1, 2)), NewList(NewVector(1, 2))},
		// empty lazy seqs are equal
		{NewLazySeq(func() interface{} { return nil }), NewLazySeq(func() interface{} { return nil })},
		{NewList(1, 2), NewLongRange(0, 3, 1).Next()},
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

func TestEqualsCharFastPath(t *testing.T) {
	for _, test := range []struct {
		name  string
		left  any
		right any
		want  bool
	}{
		{name: "same", left: NewChar('A'), right: NewChar('A'), want: true},
		{name: "different", left: NewChar('A'), right: NewChar('C')},
		{name: "char-left", left: NewChar('A'), right: int64('A')},
		{name: "char-right", left: int64('A'), right: NewChar('A')},
		{
			name:  "custom-left",
			left:  charEqualser{},
			right: NewChar('A'),
			want:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Equals(test.left, test.right); got != test.want {
				t.Fatalf(
					"Equals(%#v, %#v) = %v, want %v",
					test.left,
					test.right,
					got,
					test.want,
				)
			}
		})
	}
}
