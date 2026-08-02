package streams

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type closeTrackingReader struct {
	*bytes.Reader
	closed bool
}

func (r *closeTrackingReader) Close() error {
	r.closed = true
	return nil
}

func TestStringReaderReadLine(t *testing.T) {
	reader := NewStringReader("first\r\nsecond").(*StringReader)
	if got, want := reader.ReadLine(), "first"; got != want {
		t.Fatalf("first ReadLine = %v, want %v", got, want)
	}
	if got, want := reader.ReadLine(), "second"; got != want {
		t.Fatalf("second ReadLine = %v, want %v", got, want)
	}
	if got := reader.ReadLine(); got != nil {
		t.Fatalf("EOF ReadLine = %v, want nil", got)
	}
}

func TestPushbackReaderRunes(t *testing.T) {
	reader := NewPushbackReader(NewStringReader("ab")).(*PushbackReader)
	r, _, err := reader.ReadRune()
	if err != nil || r != 'a' {
		t.Fatalf("ReadRune = %q, %v", r, err)
	}
	if err := reader.UnreadRune(); err != nil {
		t.Fatal(err)
	}
	r, _, err = reader.ReadRune()
	if err != nil || r != 'a' {
		t.Fatalf("ReadRune after unread = %q, %v", r, err)
	}
}

func TestPushbackReaderClosesWrappedReader(t *testing.T) {
	wrapped := &closeTrackingReader{Reader: bytes.NewReader([]byte("a"))}
	reader := NewPushbackReader(wrapped).(*PushbackReader)
	reader.Close()
	if !wrapped.closed {
		t.Fatal("Close did not close wrapped reader")
	}
}

func TestFileInputStreamReadLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(path, []byte("first\nsecond"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := NewFileInputStream(path).(*FileInputStream)
	defer reader.Close()
	if got, want := reader.ReadLine(), "first"; got != want {
		t.Fatalf("first ReadLine = %v, want %v", got, want)
	}
	if got, want := reader.ReadLine(), "second"; got != want {
		t.Fatalf("second ReadLine = %v, want %v", got, want)
	}
}
