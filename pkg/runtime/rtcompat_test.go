package runtime

import (
	"strings"
	"testing"
)

func TestSubsUsesCharacterOffsets(t *testing.T) {
	const s = "a֎🙂z"

	tests := []struct {
		name  string
		start int
		end   int
		want  string
	}{
		{name: "suffix", start: 1, end: -1, want: "֎🙂z"},
		{name: "unicode range", start: 1, end: 3, want: "֎🙂"},
		{name: "empty end", start: 4, end: -1, want: ""},
		{name: "empty range", start: 2, end: 2, want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got string
			if test.end < 0 {
				got = RT.Subs(s, test.start)
			} else {
				got = RT.SubsEnd(s, test.start, test.end)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestSubsFindsUnicodeAfterASCIIBlock(t *testing.T) {
	const s = "abcdefgh🙂z"
	if got := RT.SubsEnd(s, 8, 9); got != "🙂" {
		t.Fatalf("got %q, want emoji", got)
	}
	if got := RT.Subs(s, 9); got != "z" {
		t.Fatalf("got %q, want z", got)
	}
}

func BenchmarkSubsASCII(b *testing.B) {
	s := strings.Repeat("abcdefghij", 1000)
	b.ReportAllocs()
	for b.Loop() {
		_ = RT.SubsEnd(s, 100, 9000)
	}
}
