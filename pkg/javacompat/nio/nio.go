// Package nio exposes the small java.nio charset surface used by portable
// Clojure libraries.
package nio

import (
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type Charset struct{ name string }
type CharsetDecoder struct{ charset *Charset }
type ByteBuffer struct{ bytes []byte }
type CodingErrorAction string
type CharacterCodingException struct{ message string }

var (
	UTF8                       = &Charset{name: "UTF-8"}
	USASCII                    = &Charset{name: "US-ASCII"}
	ISO88591                   = &Charset{name: "ISO-8859-1"}
	REPORT   CodingErrorAction = "REPORT"
)

func (c *Charset) String() string                                     { return c.name }
func (c *Charset) NewDecoder() *CharsetDecoder                        { return &CharsetDecoder{charset: c} }
func (d *CharsetDecoder) OnMalformedInput(_ any) *CharsetDecoder      { return d }
func (d *CharsetDecoder) OnUnmappableCharacter(_ any) *CharsetDecoder { return d }

func (d *CharsetDecoder) Decode(buffer any) string {
	data := byteSlice(buffer)
	switch d.charset.name {
	case "UTF-8":
		if !utf8.Valid(data) {
			panic(&CharacterCodingException{message: "Input length = 1"})
		}
		return string(data)
	case "US-ASCII":
		for _, value := range data {
			if value > 0x7f {
				panic(&CharacterCodingException{message: "Unmappable character"})
			}
		}
		return string(data)
	case "ISO-8859-1":
		runes := make([]rune, len(data))
		for index, value := range data {
			runes[index] = rune(value)
		}
		return string(runes)
	default:
		panic(fmt.Errorf("unsupported charset %s", d.charset.name))
	}
}

func (*CharacterCodingException) Error() string { return "Input length = 1" }
func Wrap(value any) *ByteBuffer                { return &ByteBuffer{bytes: byteSlice(value)} }

// Link gives embedders an explicit package-retention reference.
func Link() {}

func init() {
	pkgmap.SetHostClassPackage("Charset", "java.nio.charset")
	pkgmap.SetHostClass("Charset",
		lang.NewClass(reflect.TypeOf((*Charset)(nil)), "java.nio.charset.Charset"))
	for _, prefix := range []string{"Charset", "java.nio.charset.Charset"} {
		pkgmap.Set(prefix+".forName", lang.FnFunc1(func(value any) any {
			switch fmt.Sprint(value) {
			case "UTF-8", "UTF8":
				return UTF8
			case "US-ASCII", "ASCII":
				return USASCII
			case "ISO-8859-1":
				return ISO88591
			default:
				panic(fmt.Errorf("unsupported charset %v", value))
			}
		}))
	}

	pkgmap.SetHostClassPackage("ByteBuffer", "java.nio")
	pkgmap.SetHostClass("ByteBuffer",
		lang.NewClass(reflect.TypeOf((*ByteBuffer)(nil)), "java.nio.ByteBuffer"))
	for _, prefix := range []string{"ByteBuffer", "java.nio.ByteBuffer"} {
		pkgmap.Set(prefix+".wrap", lang.FnFunc1(func(value any) any { return Wrap(value) }))
	}

	pkgmap.SetHostClassPackage("StandardCharsets", "java.nio.charset")
	pkgmap.SetHostClass("StandardCharsets",
		lang.NewClass(reflect.TypeOf((*Charset)(nil)), "java.nio.charset.StandardCharsets"))
	for _, prefix := range []string{"StandardCharsets", "java.nio.charset.StandardCharsets"} {
		pkgmap.Set(prefix+".UTF_8", UTF8)
		pkgmap.Set(prefix+".US_ASCII", USASCII)
		pkgmap.Set(prefix+".ISO_8859_1", ISO88591)
	}

	pkgmap.SetHostClassPackage("CodingErrorAction", "java.nio.charset")
	pkgmap.SetHostClass("CodingErrorAction",
		lang.NewClass(reflect.TypeOf(CodingErrorAction("")), "java.nio.charset.CodingErrorAction"))
	for _, prefix := range []string{"CodingErrorAction", "java.nio.charset.CodingErrorAction"} {
		pkgmap.Set(prefix+".REPORT", REPORT)
	}

	pkgmap.SetHostClassPackage("CharacterCodingException", "java.nio.charset")
	pkgmap.SetHostClass("CharacterCodingException",
		lang.NewClass(reflect.TypeOf((*CharacterCodingException)(nil)),
			"java.nio.charset.CharacterCodingException"))
}

func byteSlice(value any) []byte {
	switch value := value.(type) {
	case *ByteBuffer:
		return value.bytes
	case []byte:
		return value
	case []int8:
		result := make([]byte, len(value))
		for index, item := range value {
			result[index] = byte(item)
		}
		return result
	case string:
		return []byte(value)
	default:
		panic(fmt.Errorf("cannot coerce %T to byte[]", value))
	}
}
