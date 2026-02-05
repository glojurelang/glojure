package lang

import (
	"testing"
)

func TestCanSeq(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		// Should return true for seqable types
		{"nil", nil, true},
		{"string", "hello", true},
		{"empty string", "", true},
		{"slice", []int{1, 2, 3}, true},
		{"empty slice", []int{}, true},
		{"array", [3]int{1, 2, 3}, true},
		{"map", map[string]int{"a": 1}, true},
		{"empty map", map[string]int{}, true},
		{"empty list", emptyList, true},
		{"lazy seq", NewLazySeq(func() interface{} { return nil }), true},

		// Should return false for non-seqable types
		{"int", 42, false},
		{"float", 3.14, false},
		{"bool", true, false},
		{"struct", struct{ X int }{X: 1}, false},
		{"pointer to int", new(int), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanSeq(tt.value)
			if result != tt.expected {
				t.Errorf("CanSeq(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestCanSeqConsistentWithSeq(t *testing.T) {
	// CanSeq should return true for any value that Seq() doesn't panic on
	seqableValues := []interface{}{
		nil,
		"test",
		[]int{1, 2, 3},
		[2]string{"a", "b"},
		map[string]int{"x": 1},
		emptyList,
		NewLazySeq(func() interface{} { return nil }),
	}

	for _, val := range seqableValues {
		if !CanSeq(val) {
			t.Errorf("CanSeq returned false for value that should be seqable: %v", val)
		}
	}
}
