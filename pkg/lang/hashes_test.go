package lang

import (
	"fmt"
	"testing"
)

func TestHashEquivalences(t *testing.T) {
	// test cases are sets of values that should hash to the same value
	testCases := [][]any{
		{nil, uint32(0)},
		{NewList(NewKeyword("a"), NewKeyword("b")), NewVector(NewKeyword("a"), NewKeyword("b"))},
		{NewList(), NewVector()},
		{NewMap(NewKeyword("a"), NewKeyword("b")), NewPersistentHashMap(NewKeyword("a"), NewKeyword("b"))},
		{NewMap(), NewPersistentHashMap()},
	}

	for i, group := range testCases {
		group := group // capture range variable
		t.Run(fmt.Sprintf("group_%d", i), func(t *testing.T) {
			if len(group) < 2 {
				t.Fatalf("test case must have at least two elements")
			}
			expectedHash := Hash(group[0])
			for _, v := range group[1:] {
				h := Hash(v)
				if h != expectedHash {
					t.Errorf("Hash(%v [%T]) = %d; want %d", v, v, h, expectedHash)
				}
			}
		})
	}
}
