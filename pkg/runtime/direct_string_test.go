package runtime

import "testing"

type countedStringer struct {
	calls *int
	text  string
}

func (s countedStringer) String() string {
	(*s.calls)++
	return s.text
}

func TestConcatStringPartsConvertsAtFinishInOrder(t *testing.T) {
	calls := 0
	parts := []any{
		"a",
		nil,
		countedStringer{calls: &calls, text: "b"},
	}
	if calls != 0 {
		t.Fatal("constructing parts converted a value")
	}
	if got := ConcatStringParts(parts); got != "ab" {
		t.Fatalf("concatenated parts = %q, want %q", got, "ab")
	}
	if calls != 1 {
		t.Fatalf("String calls = %d, want 1", calls)
	}
}
