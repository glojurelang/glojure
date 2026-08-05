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

func TestColorSyntax(t *testing.T) {
	input := `(defn greet [name]
  (println "Hello," name :from 42 true)
  ; comment
  [})`
	got := ColorSyntax([]rune(input), nil)

	for _, want := range []string{
		rainbowColors[0] + "(" + colorReset,
		rainbowColors[1] + "[" + colorReset,
		colorBoldYellow + "defn" + colorReset,
		colorGreen + `"Hello,"` + colorReset,
		colorCyan + ":from" + colorReset,
		colorMagenta + "42" + colorReset,
		colorMagenta + "true" + colorReset,
		colorGray + "; comment" + colorReset,
		colorMismatch + "}" + colorReset,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ColorSyntax() missing %q in:\n%q", want, got)
		}
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

func TestRemoteDocHasDetail(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "qualified name only",
			value: "clojure.core/println\n",
			want:  false,
		},
		{
			name:  "arglists",
			value: "clojure.core/println\n([x])",
			want:  true,
		},
		{
			name:  "docstring",
			value: "clojure.core/println\n  Prints a value",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := remoteDocHasDetail(tt.value)
			if got != tt.want {
				t.Fatalf("remoteDocHasDetail(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestShareURL(t *testing.T) {
	expressions := []string{"(def a 1)", "(+ a 2)"}
	got, err := shareURL(expressions, defaultShareBaseURL)
	if err != nil {
		t.Fatalf("shareURL() error = %v", err)
	}

	want := "https://gloathub.org/repl#s:WyIoZGVmIGEgMSkiLCIoKyBhIDIpIl0="
	if got != want {
		t.Fatalf("shareURL() = %q, want %q", got, want)
	}

	got, err = shareURL(expressions, "https://gobb.site/repl/")
	if err != nil {
		t.Fatalf("shareURL() custom base error = %v", err)
	}
	want = "https://gobb.site/repl/#s:WyIoZGVmIGEgMSkiLCIoKyBhIDIpIl0="
	if got != want {
		t.Fatalf("shareURL() custom base = %q, want %q", got, want)
	}
}

func TestWithShareBaseURL(t *testing.T) {
	var opts options
	WithShareBaseURL("https://gobb.site/repl/")(&opts)
	if opts.shareBaseURL != "https://gobb.site/repl/" {
		t.Fatalf("WithShareBaseURL() set %q", opts.shareBaseURL)
	}
}

func TestShareExpressionsFromURL(t *testing.T) {
	url, err := shareURL(
		[]string{"(def a 1)", "(+ a 2)"}, defaultShareBaseURL)
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
