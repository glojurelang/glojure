package lang

import "testing"

func TestRuneFromOctalCharLiteral(t *testing.T) {
	tests := []struct {
		literal string
		want    rune
	}{
		{`\o0`, 0},
		{`\o013`, 11},
		{`\o377`, 255},
	}

	for _, test := range tests {
		got, err := RuneFromCharLiteral(test.literal)
		if err != nil {
			t.Fatalf("RuneFromCharLiteral(%q): %v", test.literal, err)
		}
		if got != test.want {
			t.Errorf("RuneFromCharLiteral(%q) = %d, want %d", test.literal, got, test.want)
		}
	}
}

func TestRuneFromInvalidOctalCharLiteral(t *testing.T) {
	for _, literal := range []string{`\o400`, `\o1234`, `\o89`} {
		if _, err := RuneFromCharLiteral(literal); err == nil {
			t.Errorf("RuneFromCharLiteral(%q) unexpectedly succeeded", literal)
		}
	}
}
