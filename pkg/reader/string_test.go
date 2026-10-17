package reader

import (
	"strconv"
	"strings"
	"testing"
)

func TestStringReaderEscapesAndLongValues(t *testing.T) {
	long := strings.Repeat("abc雪😀", 10000)
	for _, test := range []struct{ input, want string }{
		{`""`, ""},
		{`"plain 雪😀"`, "plain 雪😀"},
		{`"` + long + `"`, long},
		{`"a\n\t\r\b\f\\\"b"`, "a\n\t\r\b\f\\\"b"},
		{"\"a\nb\"", "a\nb"},
		{`"\uD83D\uDE00"`, "😀"},
		{`"\u96EA"`, "雪"},
	} {
		got, err := New(strings.NewReader(test.input)).ReadOne()
		if err != nil || got != test.want {
			t.Fatalf("string with %d input bytes: got %q, %v", len(test.input), got, err)
		}
	}
}

func TestStringReaderPreservesInvalidStringErrors(t *testing.T) {
	for _, input := range []string{
		`"unterminated`, `"trailing\`, `"\q"`, `"\uD800"`,
	} {
		if _, err := New(strings.NewReader(input)).ReadOne(); err == nil {
			t.Fatalf("accepted invalid string %q", input)
		}
	}
}

func TestStringReaderPreservesRawControlCharacters(t *testing.T) {
	for value := rune(0); value < ' '; value++ {
		input := "\"raw" + string(value) + "character\""
		want, wantErr := strconv.Unquote(strings.ReplaceAll(input, "\n", `\n`))
		got, gotErr := New(strings.NewReader(input)).ReadOne()
		if (gotErr == nil) != (wantErr == nil) || gotErr == nil && got != want {
			t.Fatalf("control %U: got %q, %v; want %q, %v", value, got, gotErr, want, wantErr)
		}
	}
}
