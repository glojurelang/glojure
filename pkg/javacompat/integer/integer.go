// Package integer exposes JVM-faithful java.lang.Integer equivalents for
// code running on glojure. Each symbol is published two ways:
//
//   - as a Go package-level value (used when gloat AOT-compiles a Clojure
//     call site to a direct Go reference such as `compatinteger.ParseInt`);
//     and
//   - through glojure's pkgmap (used by the REPL and any dynamic
//     resolution path).
//
// Where the JVM signature is overloaded by arity (parseInt(s) /
// parseInt(s, radix), toString(i) / toString(i, radix), valueOf(int) /
// valueOf(String)), the bridge dispatches polymorphically at call time and
// returns int32 to match Java's int width.
package integer

import (
	"fmt"
	"reflect"

	jint "github.com/gloathub/gojava/integer"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

const pkg = "github.com/glojurelang/glojure/pkg/javacompat/integer"

const (
	MIN_VALUE = jint.MIN_VALUE
	MAX_VALUE = jint.MAX_VALUE
	SIZE      = jint.SIZE
	BYTES     = jint.BYTES
)

// ParseInt parses one or two args (string, optional radix) into int32.
func ParseInt(args ...any) any {
	switch len(args) {
	case 1:
		n, err := jint.ParseInt(toString(args[0]))
		if err != nil {
			panic(err.Error())
		}
		return n
	case 2:
		n, err := jint.ParseIntRadix(toString(args[0]), int(toInt32(args[1])))
		if err != nil {
			panic(err.Error())
		}
		return n
	}
	panic(fmt.Sprintf("Integer/parseInt: wrong number of args (%d)", len(args)))
}

// ValueOf accepts a string (parsed as decimal) or a number (coerced to int32).
func ValueOf(x any) int32 {
	switch v := x.(type) {
	case string:
		n, err := jint.ParseInt(v)
		if err != nil {
			panic(err.Error())
		}
		return n
	default:
		return toInt32(v)
	}
}

// ToString dispatches on arity: 1-arg returns decimal; 2-arg uses the radix.
func ToString(args ...any) string {
	switch len(args) {
	case 1:
		return jint.ToString(toInt32(args[0]))
	case 2:
		return jint.ToStringRadix(toInt32(args[0]), int(toInt32(args[1])))
	}
	panic(fmt.Sprintf("Integer/toString: wrong number of args (%d)", len(args)))
}

func ToBinaryString(x any) string { return jint.ToBinaryString(toInt32(x)) }
func ToOctalString(x any) string  { return jint.ToOctalString(toInt32(x)) }
func ToHexString(x any) string    { return jint.ToHexString(toInt32(x)) }

func BitCount(x any) int32              { return jint.BitCount(toInt32(x)) }
func NumberOfLeadingZeros(x any) int32  { return jint.NumberOfLeadingZeros(toInt32(x)) }
func NumberOfTrailingZeros(x any) int32 { return jint.NumberOfTrailingZeros(toInt32(x)) }
func HighestOneBit(x any) int32         { return jint.HighestOneBit(toInt32(x)) }
func LowestOneBit(x any) int32          { return jint.LowestOneBit(toInt32(x)) }
func Reverse(x any) int32               { return jint.Reverse(toInt32(x)) }
func ReverseBytes(x any) int32          { return jint.ReverseBytes(toInt32(x)) }
func Signum(x any) int32                { return jint.Signum(toInt32(x)) }

func Compare(x, y any) int32 { return jint.Compare(toInt32(x), toInt32(y)) }
func Max(a, b any) int32     { return jint.Max(toInt32(a), toInt32(b)) }
func Min(a, b any) int32     { return jint.Min(toInt32(a), toInt32(b)) }
func Sum(a, b any) int32     { return jint.Sum(toInt32(a), toInt32(b)) }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Integer."+jvmName, v)
	pkgmap.SetHostClassPackage("Integer", "java.lang")
	pkgmap.SetHostClass("Integer", lang.NewClassWithTypes(
		"java.lang.Integer",
		reflect.TypeOf(int(0)),
		reflect.TypeOf(int32(0)),
	))
}

func init() {
	constructor := lang.FnFunc(func(args ...any) any {
		if len(args) != 1 {
			panic(fmt.Sprintf("Integer constructor: wrong number of args (%d)", len(args)))
		}
		return ValueOf(args[0])
	})
	lang.RegisterHostConstructor("java.lang.Integer", constructor)
	lang.RegisterHostTypeConstructor(reflect.TypeOf(int(0)), constructor)
	lang.RegisterHostTypeConstructor(reflect.TypeOf(int32(0)), constructor)

	register("MIN_VALUE", "MIN_VALUE", MIN_VALUE)
	register("MAX_VALUE", "MAX_VALUE", MAX_VALUE)
	register("SIZE", "SIZE", int32(SIZE))
	register("BYTES", "BYTES", int32(BYTES))

	register("parseInt", "ParseInt", lang.FnFunc(func(args ...any) any { return ParseInt(args...) }))
	register("parseUnsignedInt", "ParseUnsignedInt", lang.FnFunc(func(args ...any) any {
		n, err := jint.ParseUnsignedInt(toString(args[0]))
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
	case int16:
		return int32(v)
	case int8:
		return int32(v)
	case uint32:
		return int32(v)
	case uint16:
		return int32(v)
	case uint8:
		return int32(v)
	case float64:
		return int32(v)
	case float32:
		return int32(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
}
