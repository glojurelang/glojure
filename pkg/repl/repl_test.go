//go:build !wasm

package repl

import "testing"

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

func TestShareClipboardText(t *testing.T) {
	got := shareClipboardText([]string{"(def a 1)", "(+ a 2)"})
	want := "(def a 1)\n\n(+ a 2)"
	if got != want {
		t.Fatalf("shareClipboardText() = %q, want %q", got, want)
	}
}
