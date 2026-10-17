package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeStringEscapeReplacementSemantics(t *testing.T) {
	replacements := lang.NewMap(
		lang.NewChar('a'), "",
		lang.NewChar('b'), false,
		lang.NewChar('c'), int64(0),
		lang.NewChar('d'), lang.NewChar('@'),
		lang.NewChar('\n'), `\n`,
		lang.NewChar('雪'), "snow",
	)
	for _, test := range []struct {
		input, want string
		mapping     any
	}{
		{"", "", nil},
		{"abc", "abc", nil},
		{"abcd\n雪😀", "b0@\\nsnow😀", replacements},
		{"unescaped 雪😀", "unescaped 雪😀", lang.NewMap()},
		{"\xff", "�", nil},
	} {
		if got := nativeStringEscape(test.input, test.mapping); got != test.want {
			t.Fatalf("escape %q = %q, want %q", test.input, got, test.want)
		}
	}
}
