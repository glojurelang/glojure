package lang

import (
	"strings"
	"testing"
)

func TestAppendWriterUnicode(t *testing.T) {
	values := []any{
		rune('é'), Char('é'), rune('雪'), Char('☃'), Char('😀'),
	}
	for _, value := range values {
		var output strings.Builder
		AppendWriter(&output, value)
		var want string
		switch value := value.(type) {
		case rune:
			want = string(value)
		case Char:
			want = string(rune(value))
		}
		if output.String() != want {
			t.Fatalf("%T: got %q, want %q", value, output.String(), want)
		}
	}
}
