// Package double exposes JVM-faithful java.lang.Double equivalents for code
// running on glojure. The structure mirrors the integer/long packages; all
// values are typed as float64 to match Java's double width.
package double

import (
	"fmt"
	"reflect"

	jdouble "github.com/gloathub/gojava/double"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/double"

const (
	SIZE  = jdouble.SIZE
	BYTES = jdouble.BYTES
)

var (
	MIN_VALUE         = jdouble.MIN_VALUE
	MAX_VALUE         = jdouble.MAX_VALUE
	MIN_NORMAL        = jdouble.MIN_NORMAL
	POSITIVE_INFINITY = jdouble.POSITIVE_INFINITY
	NEGATIVE_INFINITY = jdouble.NEGATIVE_INFINITY
	NaN               = jdouble.NaN
)

func ParseDouble(x any) float64 {
	n, err := jdouble.ParseDouble(toString(x))
	if err != nil {
		panic(err.Error())
	}
	return n
}

// ValueOf accepts a string (parsed) or any numeric (coerced to float64).
func ValueOf(x any) float64 {
	switch v := x.(type) {
	case string:
		n, err := jdouble.ParseDouble(v)
		if err != nil {
			panic(err.Error())
		}
		return n
	default:
		return toFloat64(v)
	}
}

func ToString(x any) string    { return jdouble.ToString(toFloat64(x)) }
func ToHexString(x any) string { return jdouble.ToHexString(toFloat64(x)) }

func IsNaN(x any) bool      { return jdouble.IsNaN(toFloat64(x)) }
func IsInfinite(x any) bool { return jdouble.IsInfinite(toFloat64(x)) }
func IsFinite(x any) bool   { return jdouble.IsFinite(toFloat64(x)) }

func DoubleToLongBits(x any) int64    { return jdouble.DoubleToLongBits(toFloat64(x)) }
func DoubleToRawLongBits(x any) int64 { return jdouble.DoubleToRawLongBits(toFloat64(x)) }
func LongBitsToDouble(x any) float64  { return jdouble.LongBitsToDouble(toInt64(x)) }

func Compare(x, y any) int32   { return jdouble.Compare(toFloat64(x), toFloat64(y)) }
func Max(a, b any) float64     { return jdouble.Max(toFloat64(a), toFloat64(b)) }
func Min(a, b any) float64     { return jdouble.Min(toFloat64(a), toFloat64(b)) }
func Sum(a, b any) float64     { return jdouble.Sum(toFloat64(a), toFloat64(b)) }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Double."+jvmName, v)
	pkgmap.SetHostClassPackage("Double", "java.lang")
	pkgmap.SetHostClass("Double", reflect.TypeOf(float64(0)))
}

func init() {
	register("MIN_VALUE", "MIN_VALUE", MIN_VALUE)
	register("MAX_VALUE", "MAX_VALUE", MAX_VALUE)
	register("MIN_NORMAL", "MIN_NORMAL", MIN_NORMAL)
	register("POSITIVE_INFINITY", "POSITIVE_INFINITY", POSITIVE_INFINITY)
	register("NEGATIVE_INFINITY", "NEGATIVE_INFINITY", NEGATIVE_INFINITY)
	register("NaN", "NaN", NaN)
	register("SIZE", "SIZE", int32(SIZE))
	register("BYTES", "BYTES", int32(BYTES))

	register("parseDouble", "ParseDouble", lang.FnFunc(func(args ...any) any { return ParseDouble(args[0]) }))
	register("valueOf", "ValueOf", lang.FnFunc(func(args ...any) any { return ValueOf(args[0]) }))
	register("toString", "ToString", lang.FnFunc(func(args ...any) any { return ToString(args[0]) }))
	register("toHexString", "ToHexString", lang.FnFunc(func(args ...any) any { return ToHexString(args[0]) }))

	register("isNaN", "IsNaN", lang.FnFunc(func(args ...any) any { return IsNaN(args[0]) }))
	register("isInfinite", "IsInfinite", lang.FnFunc(func(args ...any) any { return IsInfinite(args[0]) }))
	register("isFinite", "IsFinite", lang.FnFunc(func(args ...any) any { return IsFinite(args[0]) }))

	register("doubleToLongBits", "DoubleToLongBits", lang.FnFunc(func(args ...any) any { return DoubleToLongBits(args[0]) }))
	register("doubleToRawLongBits", "DoubleToRawLongBits", lang.FnFunc(func(args ...any) any { return DoubleToRawLongBits(args[0]) }))
	register("longBitsToDouble", "LongBitsToDouble", lang.FnFunc(func(args ...any) any { return LongBitsToDouble(args[0]) }))

	register("compare", "Compare", lang.FnFunc(func(args ...any) any { return Compare(args[0], args[1]) }))
	register("max", "Max", lang.FnFunc(func(args ...any) any { return Max(args[0], args[1]) }))
	register("min", "Min", lang.FnFunc(func(args ...any) any { return Min(args[0], args[1]) }))
	register("sum", "Sum", lang.FnFunc(func(args ...any) any { return Sum(args[0], args[1]) }))
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

func toFloat64(x any) float64 {
	switch v := x.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to float64", x))
}

func toInt64(x any) int64 {
	switch v := x.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int64", x))
}
