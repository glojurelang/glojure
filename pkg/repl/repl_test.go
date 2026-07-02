//go:build !wasm

package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gloathub/go-readline/inputrc"
)

func TestPrintBannerVersionPrefixes(t *testing.T) {
	var buf bytes.Buffer
	t.Setenv("GLJ_REPL_NO_BANNER", "")
	printBanner(&buf, "", "")

	out := buf.String()
	if !strings.Contains(out, " Glojure: v") {
		t.Fatalf("printBanner() missing Glojure v-prefix:\n%s", out)
	}
	if !strings.Contains(out, "\n      Go: v") {
		t.Fatalf("printBanner() missing Go v-prefix:\n%s", out)
	}
}

func TestDecodeRemoteDocValue(t *testing.T) {
	got := decodeRemoteDocValue(`"clojure.core/mapv\n([f coll])\n  Returns a vector"`)
	want := "clojure.core/mapv\n([f coll])\n  Returns a vector"
	if got != want {
		t.Fatalf("decodeRemoteDocValue() = %q, want %q", got, want)
	}

	got = decodeRemoteDocValue("nil")
	if got != "nil" {
		t.Fatalf("decodeRemoteDocValue() = %q, want nil", got)
	}
}

func TestShareURL(t *testing.T) {
	got, err := shareURL([]string{"(def a 1)", "(+ a 2)"})
	if err != nil {
		t.Fatalf("shareURL() error = %v", err)
	}

	want := "https://gloathub.org/repl#s:WyIoZGVmIGEgMSkiLCIoKyBhIDIpIl0="
	if got != want {
		t.Fatalf("shareURL() = %q, want %q", got, want)
	}
}

func TestShareExpressionsFromURL(t *testing.T) {
	url, err := shareURL([]string{"(def a 1)", "(+ a 2)"})
	if err != nil {
		t.Fatalf("shareURL() error = %v", err)
	}

	got, ok := shareExpressionsFromURL("  " + url + "  ")
	if !ok {
		t.Fatal("shareExpressionsFromURL() ok = false, want true")
	}
	want := []string{"(def a 1)", "(+ a 2)"}
	if len(got) != len(want) {
		t.Fatalf("shareExpressionsFromURL() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shareExpressionsFromURL()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if _, ok := shareExpressionsFromURL("https://gloathub.org/repl#x:nope"); ok {
		t.Fatal("shareExpressionsFromURL() ok = true, want false")
	}
}

func TestIsURLLikeInput(t *testing.T) {
	for _, input := range []string{
		"https://gloathub.org/repl#s:abc",
		"http://example.test/repl#s:abc",
		"#s:abc",
	} {
		if !isURLLikeInput(input) {
			t.Fatalf("isURLLikeInput(%q) = false, want true", input)
		}
	}

	if isURLLikeInput("(println :ok)") {
		t.Fatal("isURLLikeInput() = true, want false")
	}
}

func TestIsExplicitNewlineKey(t *testing.T) {
	for _, key := range ctrlEnterKeys() {
		if !isExplicitNewlineKey([]rune(key)) {
			t.Fatalf("isExplicitNewlineKey(%q) = false, want true", key)
		}
	}

	if isExplicitNewlineKey([]rune(inputrc.Unescape(`\C-m`))) {
		t.Fatal("isExplicitNewlineKey(C-m) = true, want false")
	}
}

func TestTrimTrailingInputText(t *testing.T) {
	got := trimTrailingInputText("  ;; comment  \n\t")
	want := "  ;; comment"
	if got != want {
		t.Fatalf("trimTrailingInputText() = %q, want %q", got, want)
	}
}

func TestSplitTopLevelForms(t *testing.T) {
	input := `
(def a 41)

(println "x y")
;; comment
(inc a)
`
	got := splitTopLevelForms(input)
	want := []string{`(def a 41)`, `(println "x y")`, `(inc a)`}
	if len(got) != len(want) {
		t.Fatalf("splitTopLevelForms() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitTopLevelForms()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestShareClipboardText(t *testing.T) {
	got := shareClipboardText([]string{"(def a 1)", "(+ a 2)"})
	want := "(def a 1)\n\n(+ a 2)"
	if got != want {
		t.Fatalf("shareClipboardText() = %q, want %q", got, want)
	}
}
