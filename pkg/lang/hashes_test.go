package lang

import (
	"fmt"
	"testing"
)

func TestHashStringMatchesHashstructureFNV1(t *testing.T) {
	for input, want := range map[string]uint32{
		"":                2216829733,
		"a":               2248259518,
		"hello":           3183334599,
		"state":           2540718800,
		"yamlstar/parser": 1399790342,
	} {
		if got := hashString(input); got != want {
			t.Errorf("hashString(%q) = %d, want %d", input, got, want)
		}
	}
}

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
