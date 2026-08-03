// Package streams exposes the most-used java.io stream classes over Go io.
package streams

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type InputStream interface{ Read([]byte) (int, error) }
type OutputStream interface{ Write([]byte) (int, error) }

type ByteArrayInputStream struct{ *bytes.Reader }
type ByteArrayOutputStream struct{ bytes.Buffer }
type StringReader struct {
	data         []byte
	position     int
	mark         int
	lastRuneSize int
	closed       bool
}
type PushbackReader struct {
	*bufio.Reader
	closer         io.Closer
	lineNumber     int
	lastRune       rune
	lastRuneSize   int
	lastRuneWasNew bool
}
type PrintWriter struct {
	io.Writer
	closer io.Closer
}
type StringWriter struct{ strings.Builder }
type FileInputStream struct {
	*os.File
	reader *bufio.Reader
}
type FileOutputStream struct{ *os.File }
type BufferedInputStream struct{ *bufio.Reader }
type BufferedOutputStream struct{ *bufio.Writer }

func (s *ByteArrayInputStream) Available() int { return s.Len() }
func (s *ByteArrayInputStream) Close()         {}
func (s *ByteArrayOutputStream) ToByteArray() []byte {
	return append([]byte(nil), s.Bytes()...)
}
func (s *ByteArrayOutputStream) ToString() string { return s.String() }
func (s *ByteArrayOutputStream) Size() int        { return s.Len() }
func (s *ByteArrayOutputStream) Close()           {}
func (s *StringReader) ensureOpen() {
	if s.closed {
		panic(errors.New("Stream closed"))
	}
}
func (s *StringReader) Close() { s.closed = true }
func (s *StringReader) Read(buffer []byte) (int, error) {
	if s.closed {
		return 0, errors.New("Stream closed")
	}
	if s.position >= len(s.data) {
		return 0, io.EOF
	}
	n := copy(buffer, s.data[s.position:])
	s.position += n
	s.lastRuneSize = 0
	return n, nil
}
func (s *StringReader) ReadRune() (rune, int, error) {
	if s.closed {
		return 0, 0, errors.New("Stream closed")
	}
	if s.position >= len(s.data) {
		return 0, 0, io.EOF
	}
	rn, size := utf8.DecodeRune(s.data[s.position:])
	s.position += size
	s.lastRuneSize = size
	return rn, size, nil
}
func (s *StringReader) UnreadRune() error {
	if s.closed {
		return errors.New("Stream closed")
	}
	if s.lastRuneSize == 0 {
		return fmt.Errorf("StringReader.UnreadRune: previous operation was not ReadRune")
	}
	s.position -= s.lastRuneSize
	s.lastRuneSize = 0
	return nil
}
func (s *StringReader) Mark(_ int) {
	s.ensureOpen()
	s.mark = s.position
}
func (s *StringReader) Reset() {
	s.ensureOpen()
	s.position = s.mark
	s.lastRuneSize = 0
}
func (s *StringReader) Skip(count any) int64 {
	s.ensureOpen()
	want := lang.MustAsInt(count)
	if want < 0 {
		panic(fmt.Sprintf("StringReader.skip: negative count (%d)", want))
	}
	remaining := len(s.data) - s.position
	if want > remaining {
		want = remaining
	}
	s.position += want
	s.lastRuneSize = 0
	return int64(want)
}
func (s *PushbackReader) Close() {
	if s.closer != nil {
		if err := s.closer.Close(); err != nil {
			panic(err)
		}
	}
}
func (s *PushbackReader) ReadRune() (rune, int, error) {
	rn, size, err := s.Reader.ReadRune()
	if err != nil {
		return rn, size, err
	}
	s.lastRune = rn
	s.lastRuneSize = size
	s.lastRuneWasNew = rn == '\n'
	if s.lastRuneWasNew {
		s.lineNumber++
	}
	return rn, size, nil
}
func (s *PushbackReader) UnreadRune() error {
	if err := s.Reader.UnreadRune(); err != nil {
		return err
	}
	if s.lastRuneWasNew {
		s.lineNumber--
	}
	s.lastRuneSize = 0
	s.lastRuneWasNew = false
	return nil
}
func (s *PushbackReader) GetLineNumber() int { return s.lineNumber }
func (w *PrintWriter) Close() {
	if w.closer != nil {
		if err := w.closer.Close(); err != nil {
			panic(err)
		}
	}
}
func (w *PrintWriter) Flush() {
	if flusher, ok := w.Writer.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			panic(err)
		}
	}
}
func (s *StringReader) ReadLine() any {
	s.ensureOpen()
	if s.position >= len(s.data) {
		return nil
	}
	remaining := s.data[s.position:]
	end := bytes.IndexByte(remaining, '\n')
	if end < 0 {
		s.position = len(s.data)
		return strings.TrimSuffix(string(remaining), "\r")
	}
	s.position += end + 1
	return strings.TrimSuffix(string(remaining[:end]), "\r")
}
func (s *FileInputStream) Read(buffer []byte) (int, error) { return s.reader.Read(buffer) }
func (s *FileInputStream) ReadLine() any                   { return readLine(s.reader) }

func readLine(reader *bufio.Reader) any {
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		panic(err)
	}
	if len(line) == 0 && err == io.EOF {
		return nil
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}
func (s *StringWriter) ToString() string { return s.String() }
func (s *StringWriter) Close()           {}

func NewByteArrayInputStream(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("ByteArrayInputStream/new: wrong number of args (%d)", len(args)))
	}
	return &ByteArrayInputStream{Reader: bytes.NewReader(byteSlice(args[0]))}
}

func NewByteArrayOutputStream(args ...any) any {
	if len(args) > 1 {
		panic(fmt.Sprintf("ByteArrayOutputStream/new: wrong number of args (%d)", len(args)))
	}
	return &ByteArrayOutputStream{}
}

func NewStringReader(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("StringReader/new: wrong number of args (%d)", len(args)))
	}
	return &StringReader{data: []byte(lang.ToString(args[0]))}
}

func NewStringWriter(args ...any) any {
	if len(args) > 1 {
		panic(fmt.Sprintf("StringWriter/new: wrong number of args (%d)", len(args)))
	}
	return &StringWriter{}
}

func NewPushbackReader(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("PushbackReader/new: wrong number of args (%d)", len(args)))
	}
	reader, ok := args[0].(io.Reader)
	if !ok {
		panic(fmt.Sprintf("PushbackReader/new: %T is not a reader", args[0]))
	}
	pushback := &PushbackReader{Reader: bufio.NewReader(reader), lineNumber: 1}
	pushback.closer, _ = args[0].(io.Closer)
	return pushback
}

func NewPrintWriter(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("PrintWriter/new: wrong number of args (%d)", len(args)))
	}
	writer, ok := args[0].(io.Writer)
	if !ok {
		panic(fmt.Sprintf("PrintWriter/new: %T is not a writer", args[0]))
	}
	result := &PrintWriter{Writer: writer}
	result.closer, _ = args[0].(io.Closer)
	return result
}

func NewFileInputStream(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("FileInputStream/new: wrong number of args (%d)", len(args)))
	}
	file, err := os.Open(lang.ToString(args[0]))
	if err != nil {
		panic(err)
	}
	return &FileInputStream{File: file, reader: bufio.NewReader(file)}
}

func NewFileOutputStream(args ...any) any {
	if len(args) < 1 || len(args) > 2 {
		panic(fmt.Sprintf("FileOutputStream/new: wrong number of args (%d)", len(args)))
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if len(args) == 2 && lang.IsTruthy(args[1]) {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	file, err := os.OpenFile(lang.ToString(args[0]), flags, 0o666)
	if err != nil {
		panic(err)
	}
	return &FileOutputStream{File: file}
}

func NewBufferedInputStream(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("BufferedInputStream/new: wrong number of args (%d)", len(args)))
	}
	reader, ok := args[0].(io.Reader)
	if !ok {
		panic(fmt.Sprintf("BufferedInputStream/new: %T is not an input stream", args[0]))
	}
	return &BufferedInputStream{Reader: bufio.NewReader(reader)}
}

func NewBufferedOutputStream(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Sprintf("BufferedOutputStream/new: wrong number of args (%d)", len(args)))
	}
	writer, ok := args[0].(io.Writer)
	if !ok {
		panic(fmt.Sprintf("BufferedOutputStream/new: %T is not an output stream", args[0]))
	}
	return &BufferedOutputStream{Writer: bufio.NewWriter(writer)}
}

func byteSlice(value any) []byte {
	switch value := value.(type) {
	case []byte:
		return value
	case string:
		return []byte(value)
	case lang.IPersistentVector:
		out := make([]byte, value.Count())
		for i := range out {
			out[i] = byte(lang.MustAsInt(value.Nth(i)))
		}
		return out
	default:
		panic(fmt.Sprintf("cannot coerce %T to byte[]", value))
	}
}

func registerClass(name, javaName string, classType reflect.Type, constructor func(...any) any) {
	pkgmap.SetHostClassPackage(name, "java.io")
	pkgmap.SetHostClass(name, lang.NewClass(classType, javaName))
	if constructor != nil {
		lang.RegisterHostConstructor(javaName,
			lang.FnFunc(func(args ...any) any { return constructor(args...) }))
	}
}

func init() {
	registerClass("InputStream", "java.io.InputStream",
		reflect.TypeOf((*InputStream)(nil)).Elem(), nil)
	registerClass("OutputStream", "java.io.OutputStream",
		reflect.TypeOf((*OutputStream)(nil)).Elem(), nil)
	registerClass("Reader", "java.io.Reader",
		reflect.TypeOf((*io.Reader)(nil)).Elem(), nil)
	registerClass("Writer", "java.io.Writer",
		reflect.TypeOf((*io.Writer)(nil)).Elem(), nil)
	registerClass("ByteArrayInputStream", "java.io.ByteArrayInputStream",
		reflect.TypeOf((*ByteArrayInputStream)(nil)), NewByteArrayInputStream)
	registerClass("ByteArrayOutputStream", "java.io.ByteArrayOutputStream",
		reflect.TypeOf((*ByteArrayOutputStream)(nil)), NewByteArrayOutputStream)
	registerClass("StringReader", "java.io.StringReader",
		reflect.TypeOf((*StringReader)(nil)), NewStringReader)
	registerClass("PushbackReader", "java.io.PushbackReader",
		reflect.TypeOf((*PushbackReader)(nil)), NewPushbackReader)
	registerClass("PrintWriter", "java.io.PrintWriter",
		reflect.TypeOf((*PrintWriter)(nil)), NewPrintWriter)
	registerClass("StringWriter", "java.io.StringWriter",
		reflect.TypeOf((*StringWriter)(nil)), NewStringWriter)
	registerClass("FileInputStream", "java.io.FileInputStream",
		reflect.TypeOf((*FileInputStream)(nil)), NewFileInputStream)
	registerClass("FileOutputStream", "java.io.FileOutputStream",
		reflect.TypeOf((*FileOutputStream)(nil)), NewFileOutputStream)
	registerClass("BufferedInputStream", "java.io.BufferedInputStream",
		reflect.TypeOf((*BufferedInputStream)(nil)), NewBufferedInputStream)
	registerClass("BufferedOutputStream", "java.io.BufferedOutputStream",
		reflect.TypeOf((*BufferedOutputStream)(nil)), NewBufferedOutputStream)
}
