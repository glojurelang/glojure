package value

import (
	"fmt"
	"math"
)

var (
	Numbers = &NumberMethods{}
)

// NumberMethods is a struct with methods that map to Clojure's Number
// class' static methods.
type NumberMethods struct{}

func (nm *NumberMethods) Add(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Add(x, y)
}

func (nm *NumberMethods) Minus(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Sub(x, y)
}

func (nm *NumberMethods) Multiply(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Multiply(x, y)
}

func (nm *NumberMethods) Divide(x, y interface{}) interface{} {
	if isNaN(x) {
		return x
	} else if isNaN(y) {
		return y
	}
	yops := Ops(y)
	if yops.IsZero(y) {
		panic("divide by zero")
	}
	return Ops(x).Combine(yops).Divide(x, y)
}

func (nm *NumberMethods) And(x, y interface{}) interface{} {
	return bitOpsCast(x) & bitOpsCast(y)
}

func (nm *NumberMethods) IsZero(x interface{}) bool {
	// convert to int64 and compare to zero
	return AsInt64(x) == 0
}

func (nm *NumberMethods) IsPos(x interface{}) bool {
	return Ops(x).IsPos(x)
}

func (nm *NumberMethods) Inc(v interface{}) interface{} {
	return nm.Add(v, 1)
}

func (nm *NumberMethods) Dec(x interface{}) interface{} {
	return nm.Add(x, -1)
}

func (nm *NumberMethods) ShiftLeft(x, y interface{}) interface{} {
	x64, y64 := bitOpsCast(x), bitOpsCast(y)
	return x64 << y64
}

func (nm *NumberMethods) ShiftRight(x, y interface{}) interface{} {
	x64, y64 := bitOpsCast(x), bitOpsCast(y)
	return x64 >> y64
}

func (nm *NumberMethods) Max(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Max(x, y)
}

func (nm *NumberMethods) Min(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Min(x, y)
}

func (nm *NumberMethods) Lt(x, y interface{}) bool {
	return Ops(x).Combine(Ops(y)).LT(x, y)
}

func (nm *NumberMethods) Gt(x, y interface{}) bool {
	return Ops(x).Combine(Ops(y)).GT(x, y)
}

func (nm *NumberMethods) Equiv(x, y interface{}) bool {
	return Ops(x).Combine(Ops(y)).Equiv(x, y)
}

func bitOpsCast(x interface{}) int64 {
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
	default:
		panic("cannot convert to int64")
	}
}

// AsNumber returns any value as a number. If the value is not a
// number, it returns false.
func AsNumber(v interface{}) (interface{}, bool) {
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
	default:
		return nil, false
	}
}

// AsInt returns any integral value as an int. If the value cannot be
// represented as an int, it returns false. Floats are not converted.
func AsInt(v interface{}) (int, bool) {
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
	default:
		return 0, false
	}
}

func IsInteger(v interface{}) bool {
	_, ok := AsInt(v)
	return ok
}

// Inc increments a number value by one. If the value is not a number,
// it returns an error.
func Inc(v interface{}) interface{} {
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
func IncP(v interface{}) interface{} {
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

type basicIntegral interface {
	int | uint | uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64
}

func incP[T basicIntegral](x T) interface{} {
	res := x + 1
	if res < x {
		return NewBigIntFromInt64(int64(x)).AddInt(1)
	}
	return res
}

func isNaN(x interface{}) bool {
	switch x := x.(type) {
	case float32:
		return math.IsNaN(float64(x))
	case float64:
		return math.IsNaN(x)
	default:
		return false
	}
}
