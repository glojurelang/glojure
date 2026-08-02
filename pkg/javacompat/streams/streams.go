// Package streams exposes the most-used java.io stream classes over Go io.
package streams

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type InputStream interface{ Read([]byte) (int, error) }
type OutputStream interface{ Write([]byte) (int, error) }

type ByteArrayInputStream struct{ *bytes.Reader }
type ByteArrayOutputStream struct{ bytes.Buffer }
type StringReader struct{ *bufio.Reader }
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
func (s *StringReader) Close()                    {}
func (s *StringReader) ReadLine() any {
	return readLine(s.Reader)
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
	return &StringReader{Reader: bufio.NewReader(strings.NewReader(lang.ToString(args[0])))}
}

func NewStringWriter(args ...any) any {
	if len(args) > 1 {
		panic(fmt.Sprintf("StringWriter/new: wrong number of args (%d)", len(args)))
	}
	return &StringWriter{}
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
