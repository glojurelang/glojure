//go:build glj_aot_runtime

package stdlib

import (
	"errors"
	"io/fs"
	"testing"
)

func TestCompactStdLibDoesNotEmbedFallbackSource(t *testing.T) {
	t.Parallel()

	_, err := StdLib.Open("clojure/core.glj")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("opening embedded source returned %v, want fs.ErrNotExist", err)
	}
}
