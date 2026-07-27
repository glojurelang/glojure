package lang

import "testing"

func TestCanSeq(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
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
		{"int", 42, false},
		{"float", 3.14, false},
		{"bool", true, false},
		{"struct", struct{ X int }{X: 1}, false},
		{"pointer to int", new(int), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanSeq(test.value); got != test.expected {
				t.Errorf("CanSeq(%v) = %v, expected %v",
					test.value, got, test.expected)
			}
		})
	}
}

func TestCanSeqConsistentWithSeq(t *testing.T) {
	seqableValues := []interface{}{
		nil,
		"test",
		[]int{1, 2, 3},
		[2]string{"a", "b"},
		map[string]int{"x": 1},
		emptyList,
		NewLazySeq(func() interface{} { return nil }),
	}

	for _, value := range seqableValues {
		if !CanSeq(value) {
			t.Errorf("CanSeq returned false for seqable value: %v", value)
		}
	}
}

func TestIsSeqTruthy(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"nil", nil, false},
		{"empty list", emptyList, false},
		{"empty vector", NewVector(), false},
		{"vector", NewVector(int64(1)), true},
		{"empty string", "", false},
		{"string", "a", true},
		{"empty slice", []int{}, false},
		{"slice", []int{1}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSeqTruthy(test.value); got != test.want {
				t.Fatalf("IsSeqTruthy(%v) = %t, want %t",
					test.value, got, test.want)
			}
		})
	}
}

func TestIsSeqTruthyRejectsCountedNonSeqableValue(t *testing.T) {
	transient := NewVector(int64(1)).AsTransient()
	defer func() {
		if recover() == nil {
			t.Fatal("IsSeqTruthy accepted a counted, non-seqable transient")
		}
	}()
	IsSeqTruthy(transient)
}
