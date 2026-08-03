package string

import "testing"

func TestStringConstructors(t *testing.T) {
	if got := New([]byte("hello"), "ASCII"); got != "hello" {
		t.Fatalf("byte String = %q, want hello", got)
	}
	if got := New([]int8{-61, -68}, "UTF-8"); got != "ü" {
		t.Fatalf("signed-byte String = %q, want ü", got)
	}
}
