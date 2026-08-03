package messagedigest

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type MessageDigest struct {
	hash hash.Hash
}

type Provider struct{ algorithms []string }
type Enumeration struct {
	values []string
	index  int
}

type javaEnumeration interface {
	HasMoreElements() bool
	NextElement() any
}

// EnumerationSeqCreate mirrors clojure.lang.EnumerationSeq/create. Java's
// implementation is lazy, but eagerly consuming the mutable Enumeration is
// observably equivalent once a caller asks for the resulting sequence.
func EnumerationSeqCreate(enumeration any) any {
	if enumeration == nil {
		return nil
	}
	javaEnumeration, ok := enumeration.(javaEnumeration)
	if !ok {
		panic(fmt.Sprintf("EnumerationSeq.create: %T is not a java.util.Enumeration", enumeration))
	}
	items := []any{}
	for javaEnumeration.HasMoreElements() {
		items = append(items, javaEnumeration.NextElement())
	}
	return lang.Seq(items)
}

func (provider *Provider) Keys() *Enumeration {
	values := make([]string, len(provider.algorithms))
	for index, algorithm := range provider.algorithms {
		values[index] = "MessageDigest." + algorithm
	}
	return &Enumeration{values: values}
}

func (enumeration *Enumeration) HasMoreElements() bool {
	return enumeration.index < len(enumeration.values)
}

func (enumeration *Enumeration) NextElement() any {
	if !enumeration.HasMoreElements() {
		panic("Enumeration has no more elements")
	}
	value := enumeration.values[enumeration.index]
	enumeration.index++
	return value
}

func GetProviders() []any {
	return []any{&Provider{algorithms: []string{"MD5", "SHA-1", "SHA-256", "SHA-384", "SHA-512"}}}
}

func GetInstance(algorithm any) any {
	var digest hash.Hash
	switch strings.ToLower(lang.ToString(algorithm)) {
	case "md5":
		digest = md5.New()
	case "sha-1", "sha1":
		digest = sha1.New()
	case "sha-256", "sha256":
		digest = sha256.New()
	case "sha-384", "sha384":
		digest = sha512.New384()
	case "sha-512", "sha512":
		digest = sha512.New()
	default:
		panic(fmt.Sprintf("MessageDigest: unsupported algorithm %s", algorithm))
	}
	return &MessageDigest{hash: digest}
}

func (digest *MessageDigest) ResolveFieldOrMethod(name string) (any, bool) {
	switch strings.ToLower(name) {
	case "reset":
		return lang.FnFunc0(func() any {
			digest.hash.Reset()
			return nil
		}), true
	case "getdigestlength":
		return lang.FnFunc0(func() any { return int64(digest.hash.Size()) }), true
	case "update":
		return lang.FnFunc(func(args ...any) any {
			if len(args) != 1 && len(args) != 3 {
				panic(fmt.Sprintf("MessageDigest.update: wrong number of args (%d)", len(args)))
			}
			data := byteSlice(args[0])
			if len(args) == 3 {
				offset, length := lang.MustAsInt(args[1]), lang.MustAsInt(args[2])
				data = data[offset : offset+length]
			}
			_, _ = digest.hash.Write(data)
			return nil
		}), true
	case "digest":
		return lang.FnFunc(func(args ...any) any {
			if len(args) > 1 {
				panic(fmt.Sprintf("MessageDigest.digest: wrong number of args (%d)", len(args)))
			}
			if len(args) == 1 {
				_, _ = digest.hash.Write(byteSlice(args[0]))
			}
			raw := digest.hash.Sum(nil)
			digest.hash.Reset()
			result := make([]int8, len(raw))
			for index, value := range raw {
				result[index] = int8(value)
			}
			return result
		}), true
	default:
		return nil, false
	}
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
	case lang.IPersistentVector:
		result := make([]byte, value.Count())
		for index := range result {
			result[index] = byte(lang.MustAsInt(value.Nth(index)))
		}
		return result
	default:
		panic(fmt.Sprintf("MessageDigest: %T is not a byte[]", value))
	}
}

func init() {
	pkgmap.Set("clojure.lang.EnumerationSeq.create", lang.FnFunc1(EnumerationSeqCreate))
	pkgmap.SetHostClassPackage("MessageDigest", "java.security")
	pkgmap.SetHostClass("MessageDigest",
		lang.NewClass(reflect.TypeOf((*MessageDigest)(nil)), "java.security.MessageDigest"))
	for _, prefix := range []string{"MessageDigest", "java.security.MessageDigest"} {
		pkgmap.Set(prefix+".getInstance", lang.FnFunc1(GetInstance))
	}
	pkgmap.SetHostClassPackage("Security", "java.security")
	pkgmap.SetHostClass("Security",
		lang.NewClass(reflect.TypeOf(struct{}{}), "java.security.Security"))
	for _, prefix := range []string{"Security", "java.security.Security"} {
		pkgmap.Set(prefix+".getProviders", lang.FnFunc0(func() any { return GetProviders() }))
	}
	pkgmap.SetHostClassPackage("Provider", "java.security")
	pkgmap.SetHostClass("Provider",
		lang.NewClass(reflect.TypeOf((*Provider)(nil)), "java.security.Provider"))
}
