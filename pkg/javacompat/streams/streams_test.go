package streams

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	if got := stream.ToByteArray(); !reflect.DeepEqual(got, []int8{65, 66, 67}) {
		t.Fatalf("bytes = %v, want ASCII ABC", got)
	}
}

func TestByteArrayInputStreamAcceptsSignedBytes(t *testing.T) {
	stream := NewByteArrayInputStream([]int8{0, 127, -128, -1}).(*ByteArrayInputStream)
	got, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0, 127, 128, 255}; !bytes.Equal(got, want) {
		t.Fatalf("bytes = %v, want %v", got, want)
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

func TestPushbackReaderAcceptsBufferSize(t *testing.T) {
	reader := NewPushbackReader(NewStringReader("ab"), int64(8)).(*PushbackReader)
	r, _, err := reader.ReadRune()
	if err != nil || r != 'a' {
		t.Fatalf("ReadRune = %q, %v", r, err)
	}
}

func TestPushbackReaderJavaReadAndUnreadOverloads(t *testing.T) {
	reader := NewPushbackReader(NewStringReader("abcd"), int64(8)).(*PushbackReader)
	readMethod, found := reader.ResolveFieldOrMethod("read")
	if !found {
		t.Fatal("read method not found")
	}
	buffer := make([]lang.Char, 4)
	if got := readMethod.(lang.IFn).Invoke(buffer, int64(0), int64(4)); got != int64(4) {
		t.Fatalf("read count = %v, want 4", got)
	}
	unreadMethod, found := reader.ResolveFieldOrMethod("unread")
	if !found {
		t.Fatal("unread method not found")
	}
	unreadMethod.(lang.IFn).Invoke(buffer, int64(1), int64(2))
	if got := readMethod.(lang.IFn).Invoke(); got != int64('b') {
		t.Fatalf("first unread character = %v, want %d", got, 'b')
	}
	if got := readMethod.(lang.IFn).Invoke(); got != int64('c') {
		t.Fatalf("second unread character = %v, want %d", got, 'c')
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

func TestPrintWriterAppend(t *testing.T) {
	var output bytes.Buffer
	writer := NewPrintWriter(&output).(*PrintWriter)
	if got := writer.Append("hello").Append(lang.NewChar('!')); got != writer {
		t.Fatal("Append did not return its receiver")
	}
	if got, want := output.String(), "hello!"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
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

func TestFileInputStreamJavaRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bytes.txt")
	if err := os.WriteFile(path, []byte{0, 127, 128, 255}, 0o600); err != nil {
		t.Fatal(err)
	}
	reader := NewFileInputStream(path).(*FileInputStream)
	defer reader.Close()
	read, ok := reader.ResolveFieldOrMethod("read")
	if !ok {
		t.Fatal("read method not resolved")
	}
	buffer := make([]int8, 4)
	if got := lang.Apply1(read, buffer); got != int64(4) {
		t.Fatalf("read = %v, want 4", got)
	}
	if want := []int8{0, 127, -128, -1}; !reflect.DeepEqual(buffer, want) {
		t.Fatalf("buffer = %v, want %v", buffer, want)
	}
	if got := lang.Apply1(read, buffer); got != int64(-1) {
		t.Fatalf("EOF read = %v, want -1", got)
	}
}

func TestStringWriterAppend(t *testing.T) {
	writer := &StringWriter{}
	if got := writer.Append("Go").Append(lang.NewChar('!')); got != writer {
		t.Fatal("Append did not return its receiver")
	}
	if got, want := writer.String(), "Go!"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestStringWriterCombinesUTF16Surrogates(t *testing.T) {
	writer := &StringWriter{}
	writer.Append(lang.Char(0xD83D)).Append(lang.Char(0xDE03))
	if got, want := writer.String(), "😃"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestColumnWriterTracksPosition(t *testing.T) {
	base := &StringWriter{}
	fields := lang.NewRef(lang.NewMap(
		lang.NewKeyword("max"), int64(72),
		lang.NewKeyword("cur"), int64(0),
		lang.NewKeyword("line"), int64(0),
		lang.NewKeyword("base"), base))
	writer := NewColumnWriter(base, int64(72), fields).(*ColumnWriter)
	write, _ := writer.ResolveFieldOrMethod("write")
	lang.Apply1(write, "abc\ndef")
	if got, want := base.String(), "abc\ndef"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
	state := fields.Deref()
	if got := lang.Get(state, lang.NewKeyword("line")); got != int64(1) {
		t.Fatalf("line = %v, want 1", got)
	}
	if got := lang.Get(state, lang.NewKeyword("cur")); got != int64(3) {
		t.Fatalf("column = %v, want 3", got)
	}
}

func TestDynamicWriterProxy(t *testing.T) {
	base := &StringWriter{}
	proxy := NewDynamicWriterProxy(lang.NewMap(
		lang.NewKeyword("write"), lang.FnFunc1(func(value any) any {
			base.WriteString(lang.ToString(value))
			return nil
		}),
		lang.NewKeyword("deref"), lang.FnFunc0(func() any { return "state" }))).(*DynamicWriterProxy)
	if _, err := io.WriteString(proxy, "hello"); err != nil {
		t.Fatal(err)
	}
	if got := base.String(); got != "hello" {
		t.Fatalf("content = %q", got)
	}
	if got := proxy.Deref(); got != "state" {
		t.Fatalf("deref = %v", got)
	}
}
