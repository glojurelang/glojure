package string

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestStringConstructors(t *testing.T) {
	if got := New([]byte("hello"), "ASCII"); got != "hello" {
		t.Fatalf("byte String = %q, want hello", got)
	}
	if got := New([]int8{-61, -68}, "UTF-8"); got != "ü" {
		t.Fatalf("signed-byte String = %q, want ü", got)
	}
}

func TestStringCharArrayRangeConstructor(t *testing.T) {
	chars := []lang.Char{'a', 'b', 'c', 'd'}
	if got := New(chars, int64(1), int64(2)); got != "bc" {
		t.Fatalf("char range String = %q, want bc", got)
	}
}
