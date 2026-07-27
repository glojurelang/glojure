package lang

import "testing"

func TestReverseStringUsesUnicodeCodePoints(t *testing.T) {
	if got, want := ReverseString("a֎"), "֎a"; got != want {
		t.Fatalf("ReverseString() = %q, want %q", got, want)
	}
}
