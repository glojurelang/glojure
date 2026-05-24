// Package uuid exposes JVM-faithful java.util.UUID equivalents for code
// running on glojure. Static methods (UUID/randomUUID, UUID/fromString,
// UUID/nameUUIDFromBytes) are published through pkgmap under both the
// bare "UUID." prefix and the fully qualified Go path. The (UUID. msb
// lsb) constructor sugar is handled by rewrite-core, which redirects to
// FromBits. Instance methods (toString, compareTo, getMostSignificantBits,
// ...) reach through lang.FieldOrMethod via reflection on *UUID.
package uuid

import (
	"fmt"
	"reflect"

	juuid "github.com/gloathub/gojava/uuid"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/uuid"

// RandomUUID mirrors UUID.randomUUID.
func RandomUUID() *juuid.UUID { return juuid.RandomUUID() }

// FromString mirrors UUID.fromString.
func FromString(args ...any) *juuid.UUID {
	if len(args) != 1 {
		panic(fmt.Sprintf("UUID/fromString: wrong number of args (%d)", len(args)))
	}
	u, err := juuid.FromString(toString(args[0]))
	if err != nil {
		panic(err.Error())
	}
	return u
}

// NameUUIDFromBytes mirrors UUID.nameUUIDFromBytes(byte[]).
func NameUUIDFromBytes(args ...any) *juuid.UUID {
	if len(args) != 1 {
		panic(fmt.Sprintf("UUID/nameUUIDFromBytes: wrong number of args (%d)", len(args)))
	}
	b, ok := toByteSlice(args[0])
	if !ok {
		panic(fmt.Sprintf("UUID/nameUUIDFromBytes: cannot coerce %T to byte[]", args[0]))
	}
	return juuid.NameUUIDFromBytes(b)
}

// FromBits is the Go target for the `(UUID. msb lsb)` constructor sugar.
// Mirrors `new UUID(long, long)`.
func FromBits(args ...any) *juuid.UUID {
	if len(args) != 2 {
		panic(fmt.Sprintf("UUID/new: wrong number of args (%d)", len(args)))
	}
	return juuid.FromBits(toInt64(args[0]), toInt64(args[1]))
}

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("UUID."+jvmName, v)
	pkgmap.SetHostClassPackage("UUID", "java.util")
	pkgmap.SetHostClass("UUID", lang.NewClass(reflect.TypeOf((*juuid.UUID)(nil)), "java.util.UUID"))
}

func init() {
	register("randomUUID", "RandomUUID", lang.FnFunc(func(_ ...any) any { return RandomUUID() }))
	register("fromString", "FromString", lang.FnFunc(func(args ...any) any { return FromString(args...) }))
	register("nameUUIDFromBytes", "NameUUIDFromBytes", lang.FnFunc(func(args ...any) any { return NameUUIDFromBytes(args...) }))
	register("fromBits", "FromBits", lang.FnFunc(func(args ...any) any { return FromBits(args...) }))
}

func toString(x any) string {
	switch v := x.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func toInt64(x any) int64 {
	switch v := x.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint64:
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int64", x))
}

func toByteSlice(x any) ([]byte, bool) {
	switch v := x.(type) {
	case []byte:
		return v, true
	case string:
		return []byte(v), true
	case []any:
		out := make([]byte, len(v))
		for i, e := range v {
			out[i] = byte(toInt64(e))
		}
		return out, true
	case lang.IPersistentVector:
		n := v.Count()
		out := make([]byte, n)
		for i := 0; i < n; i++ {
			out[i] = byte(toInt64(v.Nth(i)))
		}
		return out, true
	case lang.ISeq:
		var out []byte
		for s := v; s != nil; s = s.Next() {
			out = append(out, byte(toInt64(s.First())))
		}
		return out, true
	}
	return nil, false
}
