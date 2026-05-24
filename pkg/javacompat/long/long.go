// Package long exposes JVM-faithful java.lang.Long equivalents for code
// running on glojure. The structure mirrors the integer package; all values
// are typed as int64 to match Java's long width.
package long

import (
	"fmt"

	jlong "github.com/gloathub/gojava/long"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/long"

const (
	MIN_VALUE = jlong.MIN_VALUE
	MAX_VALUE = jlong.MAX_VALUE
	SIZE      = jlong.SIZE
	BYTES     = jlong.BYTES
)

func ParseLong(args ...any) any {
	switch len(args) {
	case 1:
		n, err := jlong.ParseLong(toString(args[0]))
		if err != nil {
			panic(err.Error())
		}
		return n
	case 2:
		n, err := jlong.ParseLongRadix(toString(args[0]), int(toInt32(args[1])))
		if err != nil {
			panic(err.Error())
		}
		return n
	}
	panic(fmt.Sprintf("Long/parseLong: wrong number of args (%d)", len(args)))
}

func ValueOf(x any) int64 {
	switch v := x.(type) {
	case string:
		n, err := jlong.ParseLong(v)
		if err != nil {
			panic(err.Error())
		}
		return n
	default:
		return toInt64(v)
	}
}

func ToString(args ...any) string {
	switch len(args) {
	case 1:
		return jlong.ToString(toInt64(args[0]))
	case 2:
		return jlong.ToStringRadix(toInt64(args[0]), int(toInt32(args[1])))
	}
	panic(fmt.Sprintf("Long/toString: wrong number of args (%d)", len(args)))
}

func ToBinaryString(x any) string { return jlong.ToBinaryString(toInt64(x)) }
func ToOctalString(x any) string  { return jlong.ToOctalString(toInt64(x)) }
func ToHexString(x any) string    { return jlong.ToHexString(toInt64(x)) }

func BitCount(x any) int32              { return jlong.BitCount(toInt64(x)) }
func NumberOfLeadingZeros(x any) int32  { return jlong.NumberOfLeadingZeros(toInt64(x)) }
func NumberOfTrailingZeros(x any) int32 { return jlong.NumberOfTrailingZeros(toInt64(x)) }
func HighestOneBit(x any) int64         { return jlong.HighestOneBit(toInt64(x)) }
func LowestOneBit(x any) int64          { return jlong.LowestOneBit(toInt64(x)) }
func Reverse(x any) int64               { return jlong.Reverse(toInt64(x)) }
func ReverseBytes(x any) int64          { return jlong.ReverseBytes(toInt64(x)) }
func Signum(x any) int32                { return jlong.Signum(toInt64(x)) }

func Compare(x, y any) int32 { return jlong.Compare(toInt64(x), toInt64(y)) }
func Max(a, b any) int64     { return jlong.Max(toInt64(a), toInt64(b)) }
func Min(a, b any) int64     { return jlong.Min(toInt64(a), toInt64(b)) }
func Sum(a, b any) int64     { return jlong.Sum(toInt64(a), toInt64(b)) }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Long."+jvmName, v)
	pkgmap.SetHostClassPackage("Long", "java.lang")
}

func init() {
	register("MIN_VALUE", "MIN_VALUE", MIN_VALUE)
	register("MAX_VALUE", "MAX_VALUE", MAX_VALUE)
	register("SIZE", "SIZE", int32(SIZE))
	register("BYTES", "BYTES", int32(BYTES))

	register("parseLong", "ParseLong", lang.FnFunc(func(args ...any) any { return ParseLong(args...) }))
	register("parseUnsignedLong", "ParseUnsignedLong", lang.FnFunc(func(args ...any) any {
		n, err := jlong.ParseUnsignedLong(toString(args[0]))
		if err != nil {
			panic(err.Error())
		}
		return n
	}))
	register("valueOf", "ValueOf", lang.FnFunc(func(args ...any) any { return ValueOf(args[0]) }))
	register("toString", "ToString", lang.FnFunc(func(args ...any) any { return ToString(args...) }))
	register("toBinaryString", "ToBinaryString", lang.FnFunc(func(args ...any) any { return ToBinaryString(args[0]) }))
	register("toOctalString", "ToOctalString", lang.FnFunc(func(args ...any) any { return ToOctalString(args[0]) }))
	register("toHexString", "ToHexString", lang.FnFunc(func(args ...any) any { return ToHexString(args[0]) }))

	register("bitCount", "BitCount", lang.FnFunc(func(args ...any) any { return BitCount(args[0]) }))
	register("numberOfLeadingZeros", "NumberOfLeadingZeros", lang.FnFunc(func(args ...any) any { return NumberOfLeadingZeros(args[0]) }))
	register("numberOfTrailingZeros", "NumberOfTrailingZeros", lang.FnFunc(func(args ...any) any { return NumberOfTrailingZeros(args[0]) }))
	register("highestOneBit", "HighestOneBit", lang.FnFunc(func(args ...any) any { return HighestOneBit(args[0]) }))
	register("lowestOneBit", "LowestOneBit", lang.FnFunc(func(args ...any) any { return LowestOneBit(args[0]) }))
	register("reverse", "Reverse", lang.FnFunc(func(args ...any) any { return Reverse(args[0]) }))
	register("reverseBytes", "ReverseBytes", lang.FnFunc(func(args ...any) any { return ReverseBytes(args[0]) }))
	register("signum", "Signum", lang.FnFunc(func(args ...any) any { return Signum(args[0]) }))

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

func toInt32(x any) int32 {
	switch v := x.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
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
