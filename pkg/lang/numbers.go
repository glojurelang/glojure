package lang

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"unicode/utf8"
)

var (
	Numbers = &NumberMethods{} // eventually make these static; this will prevent inlining
)

// NumberMethods is a struct with methods that map to Clojure's Number
// class' static methods.
type NumberMethods struct{}

func (nm *NumberMethods) UncheckedAdd(x, y any) any {
	return Ops(x).Combine(Ops(y)).UncheckedAdd(x, y)
}

func (nm *NumberMethods) UncheckedDec(x any) any {
	return Ops(x).UncheckedDec(x)
}

func (nm *NumberMethods) UncheckedIntDivide(x, y int) any {
	return x / y
}

func (nm *NumberMethods) Add(x, y any) any {
	return Ops(x).Combine(Ops(y)).Add(x, y)
}

func (nm *NumberMethods) AddP(x, y any) any {
	return Ops(x).Combine(Ops(y)).AddP(x, y)
}

func (nm *NumberMethods) Minus(x, y any) any {
	return Ops(x).Combine(Ops(y)).Sub(x, y)
}

func (nm *NumberMethods) Multiply(x, y any) any {
	return Ops(x).Combine(Ops(y)).Multiply(x, y)
}

func (nm *NumberMethods) MultiplyP(x, y any) any {
	return Ops(x).Combine(Ops(y)).MultiplyP(x, y)
}

func (nm *NumberMethods) Divide(x, y any) any {
	if isNaN(x) {
		return x
	} else if isNaN(y) {
		return y
	}
	yops := Ops(y)
	if yops.IsZero(y) {
		panic(NewArithmeticError("divide by zero"))
	}
	return Ops(x).Combine(yops).Divide(x, y)
}

func (nm *NumberMethods) Quotient(x, y any) any {
	yops := Ops(y)
	if yops.IsZero(y) {
		panic(NewArithmeticError("divide by zero"))
	}
	return Ops(x).Combine(yops).Quotient(x, y)
}

func (nm *NumberMethods) Remainder(x, y any) any {
	if isNaN(x) {
		return x
	} else if isNaN(y) {
		return y
	}
	yops := Ops(y)
	if yops.IsZero(y) {
		panic("divide by zero")
	}
	return Ops(x).Combine(yops).Remainder(x, y)
}

func (nm *NumberMethods) And(x, y any) any {
	return bitOpsCast(x) & bitOpsCast(y)
}

func IsZero(x any) bool {
	return Ops(x).IsZero(x)
}

func (nm *NumberMethods) IsZero(x any) bool {
	return IsZero(x)
}

func (nm *NumberMethods) IsPos(x any) bool {
	return Ops(x).IsPos(x)
}

func (nm *NumberMethods) IsNeg(x any) bool {
	return Ops(x).IsNeg(x)
}

func (nm *NumberMethods) Inc(v any) any {
	return nm.Add(v, 1)
}

func (nm *NumberMethods) Unchecked_inc(v any) any {
	return nm.Inc(v)
}

func (nm *NumberMethods) Dec(x any) any {
	return nm.Add(x, -1)
}

func (nm *NumberMethods) ShiftLeft(x, y any) any {
	x64, y64 := bitOpsCast(x), bitOpsCast(y)
	return x64 << (y64 & 0x3f)
}

func (nm *NumberMethods) ShiftRight(x, y any) any {
	x64, y64 := bitOpsCast(x), bitOpsCast(y)
	return x64 >> (y64 & 0x3f)
}

func (nm *NumberMethods) Max(x, y any) any {
	return Ops(x).Combine(Ops(y)).Max(x, y)
}

func (nm *NumberMethods) Min(x, y any) any {
	return Ops(x).Combine(Ops(y)).Min(x, y)
}

func (nm *NumberMethods) Lt(x, y any) bool {
	return Ops(x).Combine(Ops(y)).LT(x, y)
}

func (nm *NumberMethods) Gt(x, y any) bool {
	return Ops(x).Combine(Ops(y)).GT(x, y)
}

func (nm *NumberMethods) Lte(x, y any) bool {
	return Ops(x).Combine(Ops(y)).LTE(x, y)
}

func (nm *NumberMethods) Gte(x, y any) bool {
	return Ops(x).Combine(Ops(y)).GTE(x, y)
}

func (nm *NumberMethods) Equiv(x, y any) bool {
	return Ops(x).Combine(Ops(y)).Equiv(x, y)
}

func (nm *NumberMethods) DoubleArrayInit(size int, init any) []float64 {
	ret := make([]float64, size)
	if IsNumber(init) {
		f := AsFloat64(init)
		for i := 0; i < size; i++ {
			ret[i] = f
		}
	} else {
		s := Seq(init)
		for i := 0; i < size && s != nil; i, s = i+1, s.Next() {
			ret[i] = AsFloat64(s.First())
		}
	}
	return ret
}

func (nm *NumberMethods) ByteArray(sizeOrSeq any) []byte {
	if IsNumber(sizeOrSeq) {
		return make([]byte, MustAsInt(sizeOrSeq))
	}
	s := Seq(sizeOrSeq)
	size := Count(sizeOrSeq)
	ret := make([]byte, size)
	for i := 0; i < size && s != nil; i, s = i+1, s.Next() {
		ret[i] = AsByte(s.First())
	}
	return ret
}

func (nm *NumberMethods) ByteArrayInit(size int, init any) []byte {
	ret := make([]byte, size)
	if b, ok := init.(byte); ok {
		for i := 0; i < size; i++ {
			ret[i] = b
		}
	} else {
		s := Seq(init)
		for i := 0; i < size && s != nil; i, s = i+1, s.Next() {
			ret[i] = AsByte(s.First())
		}
	}
	return ret
}

func Abs(x any) any {
	return Ops(x).Abs(x)
}

func bitOpsCast(x any) int64 {
	switch x := x.(type) {
	case int:
		return int64(x)
	case uint:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return int64(x)
	case float32:
		return int64(x)
	case float64:
		return int64(x)
	case *BigInt:
		return x.val.Int64()
	default:
		panic(fmt.Errorf("cannot convert %T to int64", x))
	}
}

func MustAsNumber(v any) any {
	n, ok := AsNumber(v)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to number", v))
	}
	return n
}

// AsNumber returns any value as a number. If the value is not a
// number, it returns false.
func AsNumber(v any) (any, bool) {
	switch v := v.(type) {
	case int:
		return v, true
	case int64:
		return v, true
	case int32:
		return v, true
	case int16:
		return v, true
	case int8:
		return v, true
	case uint:
		return v, true
	case uint64:
		return v, true
	case uint32:
		return v, true
	case uint16:
		return v, true
	case uint8:
		return v, true
	case float64:
		return v, true
	case float32:
		return v, true
	case *BigDecimal:
		return v, true
	case *BigInt:
		return v, true
	case *Ratio:
		return v, true
	default:
		return nil, false
	}
}

func MustAsInt(v any) int {
	res, ok := AsInt(v)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to int", v))
	}
	return res
}

// AsInt returns any integral value as an int. If the value cannot be
// represented as an int, it returns false. Floats are not converted.
func AsInt(v any) (int, bool) {
	switch v := v.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case int16:
		return int(v), true
	case int8:
		return int(v), true
	case uint:
		return int(v), true
	case uint64:
		return int(v), true
	case uint32:
		return int(v), true
	case uint16:
		return int(v), true
	case uint8:
		return int(v), true
	case *BigInt:
		return int(v.val.Int64()), true
	case Char:
		return int(v), true
	default:
		return 0, false
	}
}

func AsFloat64(x any) float64 {
	switch x := x.(type) {
	case int:
		return float64(x)
	case uint:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	case *Ratio:
		f, _ := x.val.Float64()
		return f
	case *BigInt:
		// TODO: newer go versions have Int.Float64()
		return float64(x.val.Int64())
	case *big.Int:
		// TODO: newer go versions have Int.Float64()
		return float64(x.Int64())
	case *BigDecimal:
		f, _ := x.val.Float64()
		return f
	default:
		panic(fmt.Errorf("cannot convert %T to float64", x))
	}
}

var (
	byteType = reflect.TypeOf(byte(0))
)

func AsByte(x any) byte {
	switch x := x.(type) {
	case int:
		return byte(x)
	case uint:
		return byte(x)
	case int8:
		return byte(x)
	case int16:
		return byte(x)
	case int32:
		return byte(x)
	case int64:
		return byte(x)
	case uint8:
		return byte(x)
	case uint16:
		return byte(x)
	case uint32:
		return byte(x)
	case uint64:
		return byte(x)
	case float32:
		return byte(x)
	case *Ratio:
		f, _ := x.val.Float64()
		return byte(f)
	default:
		panic("cannot convert to float64")
	}
}

func IsInteger(v any) bool {
	_, ok := AsInt(v)
	return ok
}

// Inc increments a number value by one. If the value is not a number,
// it returns an error.
func Inc(v any) any {
	switch v := v.(type) {
	case int:
		return v + 1
	case int64:
		return v + 1
	case int32:
		return v + 1
	case int16:
		return v + 1
	case int8:
		return v + 1
	case uint:
		return v + 1
	case uint64:
		return v + 1
	case uint32:
		return v + 1
	case uint16:
		return v + 1
	case uint8:
		return v + 1
	case float64:
		return v + 1
	case float32:
		return v + 1
	case *BigDecimal:
		return v.AddInt(1)
	case *BigInt:
		return v.AddInt(1)
	default:
		panic(fmt.Errorf("cannot increment %T", v))
	}
}

// IncP increments a number value by one. If incrementing would
// overflow, it promotes the value to a wider type, or BigInt. If the
// value is not a number, it returns an error.
func IncP(v any) any {
	switch v := v.(type) {
	case int:
		return incP(v)
	case int64:
		return incP(v)
	case int32:
		return incP(v)
	case int16:
		return incP(v)
	case int8:
		return incP(v)
	case uint:
		return incP(v)
	case uint64:
		return incP(v)
	case uint32:
		return incP(v)
	case uint16:
		return incP(v)
	case uint8:
		return incP(v)
	case float64:
		return v + 1
	case float32:
		return v + 1
	case *BigDecimal:
		return v.AddInt(1)
	case *BigInt:
		return v.AddInt(1)
	default:
		panic(fmt.Errorf("cannot increment %T", v))
	}
}

func IsNumber(x any) bool {
	switch x.(type) {
	case int, int64, int32, int16, int8,
		uint, uint64, uint32, uint16, uint8,
		float64, float32,
		*BigDecimal, *BigInt, *Ratio:
		return true
	default:
		return false
	}
}

func BooleanCast(x any) bool {
	if b, ok := x.(bool); ok {
		return b
	}
	return !IsNil(x)
}

func ByteCast(x any) byte {
	if b, ok := x.(byte); ok {
		return b
	}
	l := AsInt64(x)
	if l < math.MinInt8 || l > math.MaxInt8 {
		panic(fmt.Errorf("value out of range for byte: %v", x))
	}
	return byte(l)
}

func CharCast(x any) Char {
	if c, ok := x.(Char); ok {
		return c
	}
	n := AsInt64(x)
	if n < 0 || n > utf8.MaxRune {
		panic(NewIllegalArgumentError(fmt.Sprintf("value out of range for char: %v", x)))
	}
	return Char(n)
}

func ShortCast(x any) int16 {
	if v, ok := x.(int16); ok {
		return v
	}
	v := AsInt64(x)
	if v < math.MinInt16 || v > math.MaxInt16 {
		panic(fmt.Errorf("value out of range for int16: %v", x))
	}
	return int16(v)
}

func FloatCast(x any) float32 {
	if v, ok := x.(float32); ok {
		return v
	}
	v := AsFloat64(x)
	if v < -math.MaxFloat32 || v > math.MaxFloat32 {
		panic(fmt.Errorf("value out of range for float32: %v", x))
	}
	return float32(v)
}

type basicIntegral interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

func incP[T basicIntegral](x T) any {
	res := x + 1
	if res < x {
		return NewBigIntFromInt64(int64(x)).AddInt(1)
	}
	return res
}

func isNaN(x any) bool {
	switch x := x.(type) {
	case float32:
		return math.IsNaN(float64(x))
	case float64:
		return math.IsNaN(x)
	default:
		return false
	}
}
