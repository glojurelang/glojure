package runtime

import (
	"strings"
	"testing"
)

func TestReadLinePreservesStream(t *testing.T) {
	input := strings.NewReader("alpha\r\n\nβeta")
	for _, expected := range []any{"alpha", "", "βeta", nil} {
		if actual := ReadLine(input); actual != expected {
			t.Fatalf("read-line = %v, want %v", actual, expected)
		}
	}
}
