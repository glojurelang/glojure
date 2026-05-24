// Package instant exposes JVM-faithful java.time.Instant equivalents for
// code running on glojure. Statics (Instant/now, Instant/parse,
// Instant/ofEpochSecond, Instant/ofEpochMilli, Instant/EPOCH) are
// published through pkgmap under both the bare "Instant." prefix and the
// fully qualified Go path. Instance methods (toString, plusMillis,
// compareTo, ...) reach through lang.FieldOrMethod via reflection on
// *Instant. java.time.Instant has no public constructor, so there is no
// (Instant. ...) sugar.
package instant

import (
	"fmt"

	jinstant "github.com/gloathub/gojava/instant"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/instant"

// EPOCH mirrors Instant.EPOCH.
var EPOCH = jinstant.EPOCH

// Now mirrors Instant.now.
func Now() *jinstant.Instant { return jinstant.Now() }

// Parse mirrors Instant.parse.
func Parse(args ...any) *jinstant.Instant {
	if len(args) != 1 {
		panic(fmt.Sprintf("Instant/parse: wrong number of args (%d)", len(args)))
	}
	out, err := jinstant.Parse(toString(args[0]))
	if err != nil {
		panic(err.Error())
	}
	return out
}

// OfEpochSecond mirrors Instant.ofEpochSecond (1+2 arg).
func OfEpochSecond(args ...any) *jinstant.Instant {
	switch len(args) {
	case 1:
		return jinstant.OfEpochSecond(toInt64(args[0]))
	case 2:
		return jinstant.OfEpochSecond(toInt64(args[0]), toInt64(args[1]))
	}
	panic(fmt.Sprintf("Instant/ofEpochSecond: wrong number of args (%d)", len(args)))
}

// OfEpochMilli mirrors Instant.ofEpochMilli.
func OfEpochMilli(args ...any) *jinstant.Instant {
	if len(args) != 1 {
		panic(fmt.Sprintf("Instant/ofEpochMilli: wrong number of args (%d)", len(args)))
	}
	return jinstant.OfEpochMilli(toInt64(args[0]))
}

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Instant."+jvmName, v)
	pkgmap.SetHostClassPackage("Instant", "java.time")
}

func init() {
	register("EPOCH", "EPOCH", EPOCH)
	register("now", "Now", lang.FnFunc(func(_ ...any) any { return Now() }))
	register("parse", "Parse", lang.FnFunc(func(args ...any) any { return Parse(args...) }))
	register("ofEpochSecond", "OfEpochSecond", lang.FnFunc(func(args ...any) any { return OfEpochSecond(args...) }))
	register("ofEpochMilli", "OfEpochMilli", lang.FnFunc(func(args ...any) any { return OfEpochMilli(args...) }))
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
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int64", x))
}
