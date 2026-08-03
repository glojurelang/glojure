package streams

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestByteArrayOutputStreamJavaWriteOverloads(t *testing.T) {
	stream := &ByteArrayOutputStream{}
	write, found := lang.FieldOrMethod(stream, "write")
	if !found {
		t.Fatal("write method not found")
	}
	lang.Apply(write, []any{int('A')})
	lang.Apply(write, []any{[]int8{'B', 'C'}})
	if got := string(stream.ToByteArray()); got != "ABC" {
		t.Fatalf("bytes = %q, want ABC", got)
	}
}

type closeTrackingReader struct {
	*bytes.Reader
	closed bool
}

type closeTrackingWriter struct {
	bytes.Buffer
	closed bool
}

func (w *closeTrackingWriter) Close() error {
	w.closed = true
	return nil
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

func TestStringReaderMarkResetSkipAndRunes(t *testing.T) {
	reader := NewStringReader("aébc").(*StringReader)
	reader.Mark(4)
	rn, _, err := reader.ReadRune()
	if err != nil || rn != 'a' {
		t.Fatalf("first ReadRune = %q, %v", rn, err)
	}
	rn, _, err = reader.ReadRune()
	if err != nil || rn != 'é' {
		t.Fatalf("second ReadRune = %q, %v", rn, err)
	}
	reader.Reset()
	if skipped := reader.Skip(1); skipped != 1 {
		t.Fatalf("Skip = %d, want 1", skipped)
	}
	rn, _, err = reader.ReadRune()
	if err != nil || rn != 'é' {
		t.Fatalf("ReadRune after reset/skip = %q, %v", rn, err)
	}
}

func TestStringReaderRejectsReadsAfterClose(t *testing.T) {
	reader := NewStringReader("a").(*StringReader)
	reader.Close()
	if _, _, err := reader.ReadRune(); err == nil || err.Error() != "Stream closed" {
		t.Fatalf("ReadRune after close error = %v", err)
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

func TestPushbackReaderTracksLineNumbers(t *testing.T) {
	reader := NewPushbackReader(NewStringReader("a\nb")).(*PushbackReader)
	if got := reader.GetLineNumber(); got != 1 {
		t.Fatalf("initial line = %d, want 1", got)
	}
	for _, want := range []rune{'a', '\n'} {
		got, _, err := reader.ReadRune()
		if err != nil || got != want {
			t.Fatalf("ReadRune = %q, %v; want %q", got, err, want)
		}
	}
	if got := reader.GetLineNumber(); got != 2 {
		t.Fatalf("line after newline = %d, want 2", got)
	}
	if err := reader.UnreadRune(); err != nil {
		t.Fatal(err)
	}
	if got := reader.GetLineNumber(); got != 1 {
		t.Fatalf("line after unread = %d, want 1", got)
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

func TestPrintWriterWritesAndCloses(t *testing.T) {
	wrapped := &closeTrackingWriter{}
	writer := NewPrintWriter(wrapped).(*PrintWriter)
	if _, err := writer.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	if got, want := wrapped.String(), "hello"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
	if !wrapped.closed {
		t.Fatal("Close did not close wrapped writer")
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
