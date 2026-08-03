// Package base64 exposes java.util.Base64 over Go's standard base64 codec.
package base64

import (
	stdbase64 "encoding/base64"
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

const pkg = "github.com/glojurelang/glojure/pkg/javacompat/base64"

type Base64 struct{}
type Encoder struct{}
type Decoder struct{}

var encoder = &Encoder{}
var decoder = &Decoder{}

func GetEncoder() *Encoder { return encoder }
func GetDecoder() *Decoder { return decoder }

func (*Encoder) EncodeToString(value any) string {
	return stdbase64.StdEncoding.EncodeToString(byteSlice(value))
}

func (*Decoder) Decode(value any) []byte {
	var text string
	switch value := value.(type) {
	case string:
		text = value
	default:
		text = string(byteSlice(value))
	}
	decoded, err := stdbase64.StdEncoding.DecodeString(text)
	if err != nil {
		panic(err)
	}
	return decoded
}

// Link gives embedders an explicit reference that retains this package and
// therefore its host-class registrations in tree-shaken builds.
func Link() {}

func init() {
	pkgmap.SetHostClassPackage("Base64", "java.util")
	pkgmap.SetHostClass("Base64",
		lang.NewClass(reflect.TypeOf(Base64{}), "java.util.Base64"))
	for _, prefix := range []string{"Base64", "java.util.Base64"} {
		pkgmap.Set(prefix+".getEncoder", lang.FnFunc0(func() any { return GetEncoder() }))
		pkgmap.Set(prefix+".getDecoder", lang.FnFunc0(func() any { return GetDecoder() }))
	}
	pkgmap.Set(pkg+".GetEncoder", lang.FnFunc0(func() any { return GetEncoder() }))
	pkgmap.Set(pkg+".GetDecoder", lang.FnFunc0(func() any { return GetDecoder() }))
}

func byteSlice(value any) []byte {
	switch value := value.(type) {
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
	case lang.IPersistentVector:
		result := make([]byte, value.Count())
		for index := range result {
			result[index] = byte(lang.MustAsInt(value.Nth(index)))
		}
		return result
	default:
		if sequence := lang.Seq(value); sequence != nil {
			result := make([]byte, 0, lang.Count(value))
			for ; sequence != nil; sequence = sequence.Next() {
				result = append(result, byte(lang.MustAsInt(sequence.First())))
			}
			return result
		}
		panic(fmt.Errorf("cannot coerce %T to byte[]", value))
	}
}
