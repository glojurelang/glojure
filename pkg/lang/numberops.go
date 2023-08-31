package lang

import (
	"fmt"
	"math"
	"math/big"
)

type Category int

const (
	CategoryInteger = iota
	CategoryFloating
	CategoryDecimal
	CategoryRatio
)

type (
	ops interface {
		Combine(y ops) ops

		IsPos(x any) bool
		IsNeg(x any) bool

		Add(x, y any) any
		// TODO: implement the precision version of Add, etc.
		AddP(x, y any) any
		UncheckedAdd(x, y any) any

		UncheckedDec(x any) any

		Sub(x, y any) any
		SubP(x, y any) any

		Multiply(x, y any) any
		MultiplyP(x, y any) any
		Divide(x, y any) any
		Quotient(x, y any) any

		Remainder(x, y any) any

		LT(x, y any) bool
		GT(x, y any) bool
		LTE(x, y any) bool
		GTE(x, y any) bool

		Max(x, y any) any
		Min(x, y any) any

		// TODO: equal vs equiv
		Equiv(x, y any) bool

		IsZero(x any) bool

		Abs(x any) any
	}
	int64Ops      struct{}
	bigIntOps     struct{}
	ratioOps      struct{}
	bigDecimalOps struct{}
	float64Ops    struct{}
)

func Ops(x any) ops {
	switch x.(type) {
	case int:
		return int64Ops{}
	case uint:
		return int64Ops{}
	case int8:
		return int64Ops{}
	case int16:
		return int64Ops{}
	case int32:
		return int64Ops{}
	case int64:
		return int64Ops{}
	case uint8:
		return int64Ops{}
	case uint16:
		return int64Ops{}
	case uint32:
		return int64Ops{}
	case uint64:
		return int64Ops{}
	case float32:
		return float64Ops{}
	case float64:
		return float64Ops{}
	case *Ratio:
		return ratioOps{}
	case *BigInt, *big.Int:
		return bigIntOps{}
	case *BigDecimal:
		return bigDecimalOps{}
	default:
		panic(fmt.Sprintf("cannot convert %T to Ops", x))
	}
}

func category(x any) Category {
	switch x.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *BigInt, *big.Int:
		return CategoryInteger
	case float32, float64:
		return CategoryFloating
	case *BigDecimal:
		return CategoryDecimal
	case *Ratio:
		return CategoryRatio
	default:
		return CategoryInteger
	}
}

func AddP(x, y any) any {
	return Ops(x).Combine(Ops(y)).AddP(x, y)
}
func Sub(x, y any) any {
	return Ops(x).Combine(Ops(y)).Sub(x, y)
}
func SubP(x, y any) any {
	return Ops(x).Combine(Ops(y)).SubP(x, y)
}
func Multiply(x, y any) any {
	return Ops(x).Combine(Ops(y)).Multiply(x, y)
}
func Divide(x, y any) any {
	return Ops(x).Combine(Ops(y)).Divide(x, y)
}
func Max(x, y any) any {
	return Ops(x).Combine(Ops(y)).Max(x, y)
}
func Min(x, y any) any {
	return Ops(x).Combine(Ops(y)).Min(x, y)
}
func NumbersEqual(x, y any) bool {
	return category(x) == category(y) &&
		Ops(x).Combine(Ops(y)).Equiv(x, y)
}

func (o int64Ops) IsPos(x any) bool {
	return AsInt64(x) > 0
}

func (o int64Ops) IsNeg(x any) bool {
	return AsInt64(x) < 0
}

func (o int64Ops) IsZero(x any) bool {
	return AsInt64(x) == 0
}

func (o int64Ops) Add(x, y any) any {
	return AsInt64(x) + AsInt64(y)
}
func (o int64Ops) AddP(x, y any) any {
	return AsInt64(x) + AsInt64(y)
}
func (o int64Ops) UncheckedAdd(x, y any) any {
	return AsInt64(x) + AsInt64(y)
}
func (o int64Ops) UncheckedDec(x any) any {
	return AsInt64(x) - 1
}
func (o int64Ops) Sub(x, y any) any {
	return AsInt64(x) - AsInt64(y)
}
func (o int64Ops) SubP(x, y any) any {
	return AsInt64(x) - AsInt64(y)
}
func (o int64Ops) Multiply(x, y any) any {
	return AsInt64(x) * AsInt64(y)
}
func (o int64Ops) MultiplyP(x, y any) any {
	xInt := AsInt64(x)
	yInt := AsInt64(y)
	if xInt == math.MinInt64 && yInt < 0 {
		return bigIntOps{}.Multiply(x, y)
	}
	ret := xInt * yInt
	if yInt != 0 && ret/yInt != xInt {
		return bigIntOps{}.Multiply(x, y)
	}
	return ret
}
func gcd(u, v int64) int64 {
	for v != 0 {
		r := u % v
		u = v
		v = r
	}
	return u
}
func (o int64Ops) Divide(x, y any) any {
	n := AsInt64(x)
	val := AsInt64(y)
	gcd := gcd(n, val)
	if gcd == 0 {
		return 0
	}
	n = n / gcd
	d := val / gcd
	if d == 1 {
		return n
	}
	if d < 0 {
		n = -n
		d = -d
	}

	return NewRatio(n, d)
}
func (o int64Ops) Quotient(x, y any) any {
	return AsInt64(x) / AsInt64(y)
}
func (o int64Ops) Remainder(x, y any) any {
	return AsInt64(x) % AsInt64(y)
}
func (o int64Ops) LT(x, y any) bool {
	return AsInt64(x) < AsInt64(y)
}
func (o int64Ops) LTE(x, y any) bool {
	return AsInt64(x) <= AsInt64(y)
}
func (o int64Ops) GT(x, y any) bool {
	return AsInt64(x) > AsInt64(y)
}
func (o int64Ops) GTE(x, y any) bool {
	return AsInt64(x) >= AsInt64(y)
}
func (o int64Ops) Max(x, y any) any {
	if AsInt64(x) > AsInt64(y) {
		return x
	}
	return y

}
func (o int64Ops) Min(x, y any) any {
	if AsInt64(x) < AsInt64(y) {
		return x
	}
	return y
}
func (o int64Ops) Equiv(x, y any) bool {
	return AsInt64(x) == AsInt64(y)
}
func (o int64Ops) Abs(x any) any {
	if AsInt64(x) < 0 {
		return -AsInt64(x)
	}
	return x
}

func (o bigIntOps) IsPos(x any) bool {
	return AsBigInt(x).val.Sign() > 0
}

func (o bigIntOps) IsNeg(x any) bool {
	return AsBigInt(x).val.Sign() < 0
}

func (o bigIntOps) IsZero(x any) bool {
	return AsBigInt(x).val.Sign() == 0
}

func (o bigIntOps) Add(x, y any) any {
	return AsBigInt(x).Add(AsBigInt(y))
}
func (o bigIntOps) AddP(x, y any) any {
	return AsBigInt(x).AddP(AsBigInt(y))
}
func (o bigIntOps) UncheckedAdd(x, y any) any {
	return AsBigInt(x).Add(AsBigInt(y))
}
func (o bigIntOps) UncheckedDec(x any) any {
	return AsBigInt(x).Sub(AsBigInt(1))
}
func (o bigIntOps) Sub(x, y any) any {
	return AsBigInt(x).Sub(AsBigInt(y))
}
func (o bigIntOps) SubP(x, y any) any {
	return AsBigInt(x).SubP(AsBigInt(y))
}
func (o bigIntOps) Multiply(x, y any) any {
	return AsBigInt(x).Multiply(AsBigInt(y))
}
func (o bigIntOps) MultiplyP(x, y any) any {
	return AsBigInt(x).Multiply(AsBigInt(y))
}
func (o bigIntOps) Divide(x, y any) any {
	return AsBigInt(x).Divide(AsBigInt(y))
}
func (o bigIntOps) Quotient(x, y any) any {
	return AsBigInt(x).Quotient(AsBigInt(y))
}
func (o bigIntOps) Remainder(x, y any) any {
	return AsBigInt(x).Remainder(AsBigInt(y))
}
func (o bigIntOps) LT(x, y any) bool {
	return AsBigInt(x).LT(AsBigInt(y))
}
func (o bigIntOps) LTE(x, y any) bool {
	return AsBigInt(x).LTE(AsBigInt(y))
}
func (o bigIntOps) GT(x, y any) bool {
	return AsBigInt(x).GT(AsBigInt(y))
}
func (o bigIntOps) GTE(x, y any) bool {
	return AsBigInt(x).GTE(AsBigInt(y))
}
func (o bigIntOps) Max(x, y any) any {
	xx := AsBigInt(x)
	yy := AsBigInt(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o bigIntOps) Min(x, y any) any {
	xx := AsBigInt(x)
	yy := AsBigInt(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y
}
func (o bigIntOps) Equiv(x, y any) bool {
	return AsBigInt(x).Cmp(AsBigInt(y)) == 0
}
func (o bigIntOps) Abs(x any) any {
	return AsBigInt(x).Abs()
}

func (o ratioOps) IsPos(x any) bool {
	return AsRatio(x).val.Sign() > 0
}

func (o ratioOps) IsNeg(x any) bool {
	return AsRatio(x).val.Sign() < 0
}

func (o ratioOps) IsZero(x any) bool {
	return AsRatio(x).val.Sign() == 0
}

func (o ratioOps) Add(x, y any) any {
	return AsRatio(x).Add(AsRatio(y))
}
func (o ratioOps) AddP(x, y any) any {
	return AsRatio(x).AddP(AsRatio(y))
}
func (o ratioOps) UncheckedAdd(x, y any) any {
	return AsRatio(x).Add(AsRatio(y))
}
func (o ratioOps) UncheckedDec(x any) any {
	return AsRatio(x).Sub(AsRatio(1))
}
func (o ratioOps) Sub(x, y any) any {
	return AsRatio(x).Sub(AsRatio(y))
}
func (o ratioOps) SubP(x, y any) any {
	return AsRatio(x).SubP(AsRatio(y))
}
func (o ratioOps) Multiply(x, y any) any {
	return AsRatio(x).Multiply(AsRatio(y))
}
func (o ratioOps) MultiplyP(x, y any) any {
	return AsRatio(x).Multiply(AsRatio(y))
}
func (o ratioOps) Divide(x, y any) any {
	return AsRatio(x).Divide(AsRatio(y))
}
func (o ratioOps) Quotient(x, y any) any {
	return AsRatio(x).Quotient(AsRatio(y))
}
func (o ratioOps) Remainder(x, y any) any {
	xRat := AsRatio(x)
	yRat := AsRatio(y)

	q := new(big.Int)
	q.Mul(xRat.val.Num(), yRat.val.Denom())

	qd := new(big.Int)
	qd.Mul(xRat.val.Denom(), yRat.val.Num())

	q.Div(q, qd)
	return Sub(x, Multiply(q, y))
}
func (o ratioOps) LT(x, y any) bool {
	return AsRatio(x).LT(AsRatio(y))
}
func (o ratioOps) LTE(x, y any) bool {
	return AsRatio(x).LTE(AsRatio(y))
}
func (o ratioOps) GT(x, y any) bool {
	return AsRatio(x).GT(AsRatio(y))
}
func (o ratioOps) GTE(x, y any) bool {
	return AsRatio(x).GTE(AsRatio(y))
}
func (o ratioOps) Max(x, y any) any {
	xx := AsRatio(x)
	yy := AsRatio(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o ratioOps) Min(x, y any) any {
	xx := AsRatio(x)
	yy := AsRatio(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y

}
func (o ratioOps) Equiv(x, y any) bool {
	return AsRatio(x).Cmp(AsRatio(y)) == 0
}
func (o ratioOps) Abs(x any) any {
	return AsRatio(x).Abs()
}

func (o bigDecimalOps) IsPos(x any) bool {
	return AsBigDecimal(x).val.Sign() > 0
}

func (o bigDecimalOps) IsNeg(x any) bool {
	return AsBigDecimal(x).val.Sign() < 0
}

func (o bigDecimalOps) IsZero(x any) bool {
	return AsBigDecimal(x).val.Sign() == 0
}

func (o bigDecimalOps) Add(x, y any) any {
	return AsBigDecimal(x).Add(AsBigDecimal(y))
}
func (o bigDecimalOps) AddP(x, y any) any {
	return AsBigDecimal(x).AddP(AsBigDecimal(y))
}
func (o bigDecimalOps) UncheckedAdd(x, y any) any {
	return AsBigDecimal(x).Add(AsBigDecimal(y))
}
func (o bigDecimalOps) UncheckedDec(x any) any {
	return AsBigDecimal(x).Sub(AsBigDecimal(1))
}
func (o bigDecimalOps) Sub(x, y any) any {
	return AsBigDecimal(x).Sub(AsBigDecimal(y))
}
func (o bigDecimalOps) SubP(x, y any) any {
	return AsBigDecimal(x).SubP(AsBigDecimal(y))
}
func (o bigDecimalOps) Multiply(x, y any) any {
	return AsBigDecimal(x).Multiply(AsBigDecimal(y))
}
func (o bigDecimalOps) MultiplyP(x, y any) any {
	return AsBigDecimal(x).Multiply(AsBigDecimal(y))
}
func (o bigDecimalOps) Divide(x, y any) any {
	return AsBigDecimal(x).Divide(AsBigDecimal(y))
}
func (o bigDecimalOps) Quotient(x, y any) any {
	return AsBigDecimal(x).Quotient(AsBigDecimal(y))
}
func (o bigDecimalOps) Remainder(x, y any) any {
	return AsBigDecimal(x).Remainder(AsBigDecimal(y))
}
func (o bigDecimalOps) LT(x, y any) bool {
	return AsBigDecimal(x).LT(AsBigDecimal(y))
}
func (o bigDecimalOps) LTE(x, y any) bool {
	return AsBigDecimal(x).LTE(AsBigDecimal(y))
}
func (o bigDecimalOps) GT(x, y any) bool {
	return AsBigDecimal(x).GT(AsBigDecimal(y))
}
func (o bigDecimalOps) GTE(x, y any) bool {
	return AsBigDecimal(x).GTE(AsBigDecimal(y))
}
func (o bigDecimalOps) Max(x, y any) any {
	xx := AsBigDecimal(x)
	yy := AsBigDecimal(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o bigDecimalOps) Min(x, y any) any {
	xx := AsBigDecimal(x)
	yy := AsBigDecimal(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y

}
func (o bigDecimalOps) Equiv(x, y any) bool {
	return AsBigDecimal(x).Cmp(AsBigDecimal(y)) == 0
}
func (o bigDecimalOps) Abs(x any) any {
	return AsBigDecimal(x).Abs()
}

func (o float64Ops) IsPos(x any) bool {
	return AsFloat64(x) > 0
}

func (o float64Ops) IsNeg(x any) bool {
	return AsFloat64(x) < 0
}

func (o float64Ops) IsZero(x any) bool {
	return AsFloat64(x) == 0
}

func (o float64Ops) Add(x, y any) any {
	return AsFloat64(x) + AsFloat64(y)
}
func (o float64Ops) AddP(x, y any) any {
	return AsFloat64(x) + AsFloat64(y)
}
func (o float64Ops) UncheckedAdd(x, y any) any {
	return AsFloat64(x) + AsFloat64(y)
}
func (o float64Ops) UncheckedDec(x any) any {
	return AsFloat64(x) - 1
}
func (o float64Ops) Sub(x, y any) any {
	return AsFloat64(x) - AsFloat64(y)
}
func (o float64Ops) SubP(x, y any) any {
	return AsFloat64(x) - AsFloat64(y)
}
func (o float64Ops) Multiply(x, y any) any {
	return AsFloat64(x) * AsFloat64(y)
}
func (o float64Ops) MultiplyP(x, y any) any {
	// as in clojure, no overflow check
	return AsFloat64(x) * AsFloat64(y)
}
func (o float64Ops) Divide(x, y any) any {
	return AsFloat64(x) / AsFloat64(y)
}
func (o float64Ops) Quotient(x, y any) any {
	xf := AsFloat64(x)
	yf := AsFloat64(y)
	if IsZero(yf) {
		panic(NewArithmeticError("divide by zero"))
	}
	q := xf / yf
	if q <= math.MaxInt64 && q >= math.MinInt64 {
		return float64(int64(q))
	}
	return AsBigDecimal(AsBigInt(q))
}
func (o float64Ops) Remainder(x, y any) any {
	return math.Mod(AsFloat64(x), AsFloat64(y))
}
func (o float64Ops) LT(x, y any) bool {
	return AsFloat64(x) < AsFloat64(y)
}
func (o float64Ops) LTE(x, y any) bool {
	return AsFloat64(x) <= AsFloat64(y)
}
func (o float64Ops) GT(x, y any) bool {
	return AsFloat64(x) > AsFloat64(y)
}
func (o float64Ops) GTE(x, y any) bool {
	return AsFloat64(x) >= AsFloat64(y)
}
func (o float64Ops) Max(x, y any) any {
	return math.Max(AsFloat64(x), AsFloat64(y))
}
func (o float64Ops) Min(x, y any) any {
	return math.Min(AsFloat64(x), AsFloat64(y))
}
func (o float64Ops) Equiv(x, y any) bool {
	return AsFloat64(x) == AsFloat64(y)
}
func (o float64Ops) Abs(x any) any {
	return math.Abs(AsFloat64(x))
}

func (o int64Ops) Combine(y ops) ops {
	switch y.(type) {
	case int64Ops:
		return o
	case bigIntOps:
		return y
	case ratioOps:
		return y
	case bigDecimalOps:
		return y
	case float64Ops:
		return y
	default:
		panic("cannot combine Ops")
	}
}
func (o bigIntOps) Combine(y ops) ops {
	switch y.(type) {
	case int64Ops:
		return o
	case bigIntOps:
		return o
	case ratioOps:
		return y
	case bigDecimalOps:
		return y
	case float64Ops:
		return y
	default:
		panic("cannot combine Ops")
	}
}
func (o ratioOps) Combine(y ops) ops {
	switch y.(type) {
	case int64Ops:
		return o
	case bigIntOps:
		return o
	case ratioOps:
		return o
	case bigDecimalOps:
		return y
	case float64Ops:
		return y
	default:
		panic("cannot combine Ops")
	}
}
func (o bigDecimalOps) Combine(y ops) ops {
	switch y.(type) {
	case int64Ops:
		return o
	case bigIntOps:
		return o
	case ratioOps:
		return o
	case bigDecimalOps:
		return o
	case float64Ops:
		return y
	default:
		panic("cannot combine Ops")
	}
}
func (o float64Ops) Combine(y ops) ops {
	switch y.(type) {
	case int64Ops:
		return o
	case bigIntOps:
		return o
	case ratioOps:
		return o
	case bigDecimalOps:
		return o
	case float64Ops:
		return o
	default:
		panic("cannot combine Ops")
	}
}
func AsInt64(x any) int64 {
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
	case *Ratio:
		n := x.Numerator()
		d := x.Denominator()
		q := n.Quo(n, d)
		return q.Int64()
	case *BigInt:
		return x.val.Int64()
	case *BigDecimal:
		i, _ := x.val.Int(nil)
		return i.Int64()
	default:
		panic(fmt.Errorf("cannot convert %T to int64", x))
	}
}

func AsBigInt(x any) *BigInt {
	switch x := x.(type) {
	case int:
		return NewBigIntFromInt64(int64(x))
	case uint:
		return NewBigIntFromInt64(int64(x))
	case int8:
		return NewBigIntFromInt64(int64(x))
	case int16:
		return NewBigIntFromInt64(int64(x))
	case int32:
		return NewBigIntFromInt64(int64(x))
	case int64:
		return NewBigIntFromInt64(int64(x))
	case uint8:
		return NewBigIntFromInt64(int64(x))
	case uint16:
		return NewBigIntFromInt64(int64(x))
	case uint32:
		return NewBigIntFromInt64(int64(x))
	case uint64:
		return NewBigIntFromInt64(int64(x))
	case float32:
		return NewBigIntFromInt64(int64(x))
	case float64:
		return NewBigIntFromInt64(int64(x))
	case *BigInt:
		return x
	case *big.Int:
		return NewBigIntFromGoBigInt(x)
	default:
		panic(fmt.Errorf("cannot convert %T to BigInt", x))
	}
}

func AsRatio(x any) *Ratio {
	switch x := x.(type) {
	case int:
		return NewRatio(int64(x), 1)
	case uint:
		return NewRatio(int64(x), 1)
	case int8:
		return NewRatio(int64(x), 1)
	case int16:
		return NewRatio(int64(x), 1)
	case int32:
		return NewRatio(int64(x), 1)
	case int64:
		return NewRatio(int64(x), 1)
	case uint8:
		return NewRatio(int64(x), 1)
	case uint16:
		return NewRatio(int64(x), 1)
	case uint32:
		return NewRatio(int64(x), 1)
	case uint64:
		return NewRatio(int64(x), 1)
	case float32:
		return NewRatio(int64(x), 1)
	case float64:
		return NewRatio(int64(x), 1)
	case *BigInt:
		return NewRatioBigInt(x, NewBigIntFromInt64(1))
	case *big.Int:
		return NewRatioBigInt(NewBigIntFromGoBigInt(x), NewBigIntFromInt64(1))
	case *Ratio:
		return x
	default:
		panic(fmt.Errorf("cannot convert %T to Ratio", x))
	}
}

func AsBigDecimal(x any) *BigDecimal {
	switch x := x.(type) {
	case int:
		return NewBigDecimalFromFloat64(float64(x))
	case uint:
		return NewBigDecimalFromFloat64(float64(x))
	case int8:
		return NewBigDecimalFromFloat64(float64(x))
	case int16:
		return NewBigDecimalFromFloat64(float64(x))
	case int32:
		return NewBigDecimalFromFloat64(float64(x))
	case int64:
		return NewBigDecimalFromFloat64(float64(x))
	case uint8:
		return NewBigDecimalFromFloat64(float64(x))
	case uint16:
		return NewBigDecimalFromFloat64(float64(x))
	case uint32:
		return NewBigDecimalFromFloat64(float64(x))
	case uint64:
		return NewBigDecimalFromFloat64(float64(x))
	case float32:
		return NewBigDecimalFromFloat64(float64(x))
	case float64:
		return NewBigDecimalFromFloat64(float64(x))
	case *BigDecimal:
		return x
	case *BigInt:
		f := new(big.Float)
		f.SetInt(x.val)
		return NewBigDecimalFromBigFloat(f)
	case *big.Int:
		f := new(big.Float)
		f.SetInt(x)
		return NewBigDecimalFromBigFloat(f)
	case *Ratio:
		f := new(big.Float)
		f.SetRat(x.val)
		return NewBigDecimalFromBigFloat(f)
	default:
		panic(fmt.Errorf("cannot convert %T to BigDecimal", x))
	}
}
