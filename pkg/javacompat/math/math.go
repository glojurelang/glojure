// Package math exposes JVM-faithful java.lang.Math equivalents for code
// running on glojure. Each symbol is published two ways:
//
//   - as a Go package-level value (used when gloat AOT-compiles a Clojure
//     call site to a direct Go reference such as `compatmath.Sqrt`); and
//   - through glojure's pkgmap (used by the REPL and any dynamic
//     resolution path).
//
// Java's Math methods are overloaded by argument type (int, long, float,
// double). The polymorphic helpers here type-switch on the runtime argument
// and dispatch to the matching gojava overload, returning a Go value whose
// type mirrors what the JVM would have produced for the same call.
package math

import (
	"fmt"
	mathrand "math/rand"
	"reflect"

	jmath "github.com/gloathub/gojava/math"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

const pkg = "github.com/glojurelang/glojure/pkg/javacompat/math"

// Math is the placeholder type registered as java.lang.Math's reflect.Type.
// java.lang.Math is final with a private constructor in the JVM, so no
// instances ever exist; the value here only needs to make
// (instance? Class Math) succeed so (ns-imports *ns*) sees the import.
type Math struct{}

// JavaRandom provides the small java.util.Random instance surface used by
// clojure.data.generators and other portable libraries.
type JavaRandom struct{ rng *mathrand.Rand }

func NewJavaRandom(args ...any) any {
	seed := int64(0)
	if len(args) == 1 {
		seed = lang.AsInt64(args[0])
	}
	if len(args) > 1 {
		panic("java.util.Random expects zero or one constructor argument")
	}
	return &JavaRandom{rng: mathrand.New(mathrand.NewSource(seed))}
}

func (r *JavaRandom) NextBoolean() bool   { return r.rng.Int63()&1 == 0 }
func (r *JavaRandom) NextDouble() float64 { return r.rng.Float64() }
func (r *JavaRandom) NextFloat() float32  { return r.rng.Float32() }
func (r *JavaRandom) NextLong() int64     { return r.rng.Int63() }

var (
	PI = jmath.PI
	E  = jmath.E
)

var (
	Sqrt      = jmath.Sqrt
	Cbrt      = jmath.Cbrt
	Sin       = jmath.Sin
	Cos       = jmath.Cos
	Tan       = jmath.Tan
	Asin      = jmath.Asin
	Acos      = jmath.Acos
	Atan      = jmath.Atan
	Sinh      = jmath.Sinh
	Cosh      = jmath.Cosh
	Tanh      = jmath.Tanh
	Log       = jmath.Log
	Log10     = jmath.Log10
	Log1p     = jmath.Log1p
	Exp       = jmath.Exp
	Expm1     = jmath.Expm1
	Ceil      = jmath.Ceil
	Floor     = jmath.Floor
	Rint      = jmath.Rint
	Signum    = jmath.Signum
	ToRadians = jmath.ToRadians
	ToDegrees = jmath.ToDegrees
)

var (
	Pow           = jmath.Pow
	Atan2         = jmath.Atan2
	CopySign      = jmath.CopySign
	Hypot         = jmath.Hypot
	IEEEremainder = jmath.IEEEremainder
)

var Round = jmath.Round

var Random = jmath.Random

func Abs(x any) any {
	switch v := x.(type) {
	case int64:
		return jmath.AbsLong(v)
	case int32:
		return jmath.AbsInt(v)
	case int:
		return jmath.AbsLong(int64(v))
	case float32:
		return float32(jmath.Abs(float64(v)))
	case float64:
		return jmath.Abs(v)
	default:
		panic(fmt.Sprintf("Math/abs: unsupported type %T", x))
	}
}

func Max(a, b any) any { return maxMin(a, b, true) }
func Min(a, b any) any { return maxMin(a, b, false) }

func FloorDiv(a, b any) any {
	if x, y, ok := bothInt32(a, b); ok {
		return jmath.FloorDivInt(x, y)
	}
	return jmath.FloorDivLong(toInt64(a), toInt64(b))
}

func FloorMod(a, b any) any {
	if x, y, ok := bothInt32(a, b); ok {
		return jmath.FloorModInt(x, y)
	}
	return jmath.FloorModLong(toInt64(a), toInt64(b))
}

func AddExact(a, b any) any {
	if x, y, ok := bothInt32(a, b); ok {
		return jmath.AddExactInt(x, y)
	}
	return jmath.AddExactLong(toInt64(a), toInt64(b))
}

func SubtractExact(a, b any) any {
	if x, y, ok := bothInt32(a, b); ok {
		return jmath.SubtractExactInt(x, y)
	}
	return jmath.SubtractExactLong(toInt64(a), toInt64(b))
}

func MultiplyExact(a, b any) any {
	if x, y, ok := bothInt32(a, b); ok {
		return jmath.MultiplyExactInt(x, y)
	}
	return jmath.MultiplyExactLong(toInt64(a), toInt64(b))
}

func NegateExact(x any) any {
	if v, ok := x.(int32); ok {
		return jmath.NegateExactInt(v)
	}
	return jmath.NegateExactLong(toInt64(x))
}

func IncrementExact(x any) any {
	if v, ok := x.(int32); ok {
		return jmath.IncrementExactInt(v)
	}
	return jmath.IncrementExactLong(toInt64(x))
}

func DecrementExact(x any) any {
	if v, ok := x.(int32); ok {
		return jmath.DecrementExactInt(v)
	}
	return jmath.DecrementExactLong(toInt64(x))
}

func ToIntExact(x any) any {
	return jmath.ToIntExact(toInt64(x))
}

// register publishes a Math symbol under both the bridge's Go import path
// (used by AOT after rewrite resolves `Math/abs` to a fully-qualified Go
// symbol) and the JVM-style "Math.<name>" form (used at the REPL, where
// EvalASTMaybeHostForm looks up `pkgmap.Get(Class + "." + name)` directly).
func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Math."+jvmName, v)
	pkgmap.SetHostClassPackage("Math", "java.lang")
	pkgmap.SetHostClass("Math", lang.NewClass(reflect.TypeOf(Math{}), "java.lang.Math"))
}

func init() {
	randomClass := lang.NewClass(
		reflect.TypeOf((*JavaRandom)(nil)), "java.util.Random")
	pkgmap.SetHostClassPackage("Random", "java.util")
	pkgmap.SetHostClass("Random", randomClass)
	lang.RegisterHostConstructor("java.util.Random", lang.FnFunc(NewJavaRandom))
	lang.RegisterHostTypeConstructor(
		reflect.TypeOf((*JavaRandom)(nil)), lang.FnFunc(NewJavaRandom))

	register("PI", "PI", PI)
	register("E", "E", E)

	register("sqrt", "Sqrt", fn1Float64(Sqrt))
	register("cbrt", "Cbrt", fn1Float64(Cbrt))
	register("sin", "Sin", fn1Float64(Sin))
	register("cos", "Cos", fn1Float64(Cos))
	register("tan", "Tan", fn1Float64(Tan))
	register("asin", "Asin", fn1Float64(Asin))
	register("acos", "Acos", fn1Float64(Acos))
	register("atan", "Atan", fn1Float64(Atan))
	register("sinh", "Sinh", fn1Float64(Sinh))
	register("cosh", "Cosh", fn1Float64(Cosh))
	register("tanh", "Tanh", fn1Float64(Tanh))
	register("log", "Log", fn1Float64(Log))
	register("log10", "Log10", fn1Float64(Log10))
	register("log1p", "Log1p", fn1Float64(Log1p))
	register("exp", "Exp", fn1Float64(Exp))
	register("expm1", "Expm1", fn1Float64(Expm1))
	register("ceil", "Ceil", fn1Float64(Ceil))
	register("floor", "Floor", fn1Float64(Floor))
	register("rint", "Rint", fn1Float64(Rint))
	register("signum", "Signum", fn1Float64(Signum))
	register("toRadians", "ToRadians", fn1Float64(ToRadians))
	register("toDegrees", "ToDegrees", fn1Float64(ToDegrees))

	register("pow", "Pow", fn2Float64(Pow))
	register("atan2", "Atan2", fn2Float64(Atan2))
	register("copySign", "CopySign", fn2Float64(CopySign))
	register("hypot", "Hypot", fn2Float64(Hypot))
	register("IEEEremainder", "IEEEremainder", fn2Float64(IEEEremainder))

	register("round", "Round", lang.FnFunc(func(args ...any) any {
		return Round(toFloat64(args[0]))
	}))
	register("random", "Random", lang.FnFunc(func(args ...any) any {
		return Random()
	}))

	register("abs", "Abs", lang.FnFunc(func(args ...any) any { return Abs(args[0]) }))
	register("max", "Max", lang.FnFunc(func(args ...any) any { return Max(args[0], args[1]) }))
	register("min", "Min", lang.FnFunc(func(args ...any) any { return Min(args[0], args[1]) }))
	register("floorDiv", "FloorDiv", lang.FnFunc(func(args ...any) any { return FloorDiv(args[0], args[1]) }))
	register("floorMod", "FloorMod", lang.FnFunc(func(args ...any) any { return FloorMod(args[0], args[1]) }))
	register("addExact", "AddExact", lang.FnFunc(func(args ...any) any { return AddExact(args[0], args[1]) }))
	register("subtractExact", "SubtractExact", lang.FnFunc(func(args ...any) any { return SubtractExact(args[0], args[1]) }))
	register("multiplyExact", "MultiplyExact", lang.FnFunc(func(args ...any) any { return MultiplyExact(args[0], args[1]) }))
	register("negateExact", "NegateExact", lang.FnFunc(func(args ...any) any { return NegateExact(args[0]) }))
	register("incrementExact", "IncrementExact", lang.FnFunc(func(args ...any) any { return IncrementExact(args[0]) }))
	register("decrementExact", "DecrementExact", lang.FnFunc(func(args ...any) any { return DecrementExact(args[0]) }))
	register("toIntExact", "ToIntExact", lang.FnFunc(func(args ...any) any { return ToIntExact(args[0]) }))
}

func fn1Float64(f func(float64) float64) lang.IFn {
	return lang.FnFunc(func(args ...any) any { return f(toFloat64(args[0])) })
}

func fn2Float64(f func(float64, float64) float64) lang.IFn {
	return lang.FnFunc(func(args ...any) any {
		return f(toFloat64(args[0]), toFloat64(args[1]))
	})
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
	case uint:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case *lang.Ratio:
		return lang.AsFloat64(v)
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

func bothInt32(a, b any) (int32, int32, bool) {
	x, ok1 := a.(int32)
	y, ok2 := b.(int32)
	return x, y, ok1 && ok2
}

func maxMin(a, b any, isMax bool) any {
	if isFloat(a) || isFloat(b) {
		x, y := toFloat64(a), toFloat64(b)
		if isMax {
			return jmath.Max(x, y)
		}
		return jmath.Min(x, y)
	}
	if ax, ay, ok := bothInt32(a, b); ok {
		if isMax {
			return jmath.MaxInt(ax, ay)
		}
		return jmath.MinInt(ax, ay)
	}
	x, y := toInt64(a), toInt64(b)
	if isMax {
		return jmath.MaxLong(x, y)
	}
	return jmath.MinLong(x, y)
}

func isFloat(x any) bool {
	switch x.(type) {
	case float64, float32:
		return true
	}
	return false
}
