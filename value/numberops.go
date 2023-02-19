package value

import (
	"fmt"
)

type (
	ops interface {
		Combine(y ops) ops

		IsPos(x interface{}) bool

		Add(x, y interface{}) interface{}
		// TODO: implement the precision version of Add, etc.
		AddP(x, y interface{}) interface{}

		Sub(x, y interface{}) interface{}
		SubP(x, y interface{}) interface{}

		Multiply(x, y interface{}) interface{}
		Divide(x, y interface{}) interface{}

		LT(x, y interface{}) bool

		Max(x, y interface{}) interface{}
		Min(x, y interface{}) interface{}

		Equal(x, y interface{}) bool

		IsZero(x interface{}) bool
	}
	int64Ops      struct{}
	bigIntOps     struct{}
	ratioOps      struct{}
	bigDecimalOps struct{}
	float64Ops    struct{}
)

func Ops(x interface{}) ops {
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
	case *BigInt:
		return bigIntOps{}
	case *BigDecimal:
		return bigDecimalOps{}
	default:
		panic(fmt.Sprintf("cannot convert %T to Ops", x))
	}
}

func AddP(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).AddP(x, y)
}
func Sub(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Sub(x, y)
}
func SubP(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).SubP(x, y)
}

func Max(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Max(x, y)
}
func Min(x, y interface{}) interface{} {
	return Ops(x).Combine(Ops(y)).Min(x, y)
}
func NumbersEqual(x, y interface{}) bool {
	return Ops(x).Combine(Ops(y)).Equal(x, y)
}

func (o int64Ops) IsPos(x interface{}) bool {
	return x.(int64) > 0
}

func (o int64Ops) IsZero(x interface{}) bool {
	return x.(int64) == 0
}

func (o int64Ops) Add(x, y interface{}) interface{} {
	return AsInt64(x) + AsInt64(y)
}
func (o int64Ops) AddP(x, y interface{}) interface{} {
	return AsInt64(x) + AsInt64(y)
}
func (o int64Ops) Sub(x, y interface{}) interface{} {
	return AsInt64(x) - AsInt64(y)
}
func (o int64Ops) SubP(x, y interface{}) interface{} {
	return AsInt64(x) - AsInt64(y)
}
func (o int64Ops) Multiply(x, y interface{}) interface{} {
	return AsInt64(x) * AsInt64(y)
}
func (o int64Ops) Divide(x, y interface{}) interface{} {
	return AsInt64(x) / AsInt64(y)
}
func (o int64Ops) LT(x, y interface{}) bool {
	return AsInt64(x) < AsInt64(y)
}
func (o int64Ops) Max(x, y interface{}) interface{} {
	if AsInt64(x) > AsInt64(y) {
		return x
	}
	return y

}
func (o int64Ops) Min(x, y interface{}) interface{} {
	if AsInt64(x) < AsInt64(y) {
		return x
	}
	return y
}
func (o int64Ops) Equal(x, y interface{}) bool {
	return AsInt64(x) == AsInt64(y)
}

func (o bigIntOps) IsPos(x interface{}) bool {
	return AsBigInt(x).val.Sign() > 0
}

func (o bigIntOps) IsZero(x interface{}) bool {
	return AsBigInt(x).val.Sign() == 0
}

func (o bigIntOps) Add(x, y interface{}) interface{} {
	return AsBigInt(x).Add(AsBigInt(y))
}
func (o bigIntOps) AddP(x, y interface{}) interface{} {
	return AsBigInt(x).AddP(AsBigInt(y))
}
func (o bigIntOps) Sub(x, y interface{}) interface{} {
	return AsBigInt(x).Sub(AsBigInt(y))
}
func (o bigIntOps) SubP(x, y interface{}) interface{} {
	return AsBigInt(x).SubP(AsBigInt(y))
}
func (o bigIntOps) Multiply(x, y interface{}) interface{} {
	return AsBigInt(x).Multiply(AsBigInt(y))
}
func (o bigIntOps) Divide(x, y interface{}) interface{} {
	return AsBigInt(x).Divide(AsBigInt(y))
}
func (o bigIntOps) LT(x, y interface{}) bool {
	return AsBigInt(x).LT(AsBigInt(y))
}
func (o bigIntOps) Max(x, y interface{}) interface{} {
	xx := AsBigInt(x)
	yy := AsBigInt(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o bigIntOps) Min(x, y interface{}) interface{} {
	xx := AsBigInt(x)
	yy := AsBigInt(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y
}
func (o bigIntOps) Equal(x, y interface{}) bool {
	return AsBigInt(x).Cmp(AsBigInt(y)) == 0
}

func (o ratioOps) IsPos(x interface{}) bool {
	return AsRatio(x).val.Sign() > 0
}

func (o ratioOps) IsZero(x interface{}) bool {
	return AsRatio(x).val.Sign() == 0
}

func (o ratioOps) Add(x, y interface{}) interface{} {
	return AsRatio(x).Add(AsRatio(y))
}
func (o ratioOps) AddP(x, y interface{}) interface{} {
	return AsRatio(x).AddP(AsRatio(y))
}
func (o ratioOps) Sub(x, y interface{}) interface{} {
	return AsRatio(x).Sub(AsRatio(y))
}
func (o ratioOps) SubP(x, y interface{}) interface{} {
	return AsRatio(x).SubP(AsRatio(y))
}
func (o ratioOps) Multiply(x, y interface{}) interface{} {
	return AsRatio(x).Multiply(AsRatio(y))
}
func (o ratioOps) Divide(x, y interface{}) interface{} {
	return AsRatio(x).Divide(AsRatio(y))
}
func (o ratioOps) LT(x, y interface{}) bool {
	return AsRatio(x).LT(AsRatio(y))
}
func (o ratioOps) Max(x, y interface{}) interface{} {
	xx := AsRatio(x)
	yy := AsRatio(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o ratioOps) Min(x, y interface{}) interface{} {
	xx := AsRatio(x)
	yy := AsRatio(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y

}
func (o ratioOps) Equal(x, y interface{}) bool {
	return AsRatio(x).Cmp(AsRatio(y)) == 0
}

func (o bigDecimalOps) IsPos(x interface{}) bool {
	return AsBigDecimal(x).val.Sign() > 0
}

func (o bigDecimalOps) IsZero(x interface{}) bool {
	return AsBigDecimal(x).val.Sign() == 0
}

func (o bigDecimalOps) Add(x, y interface{}) interface{} {
	return AsBigDecimal(x).Add(AsBigDecimal(y))
}
func (o bigDecimalOps) AddP(x, y interface{}) interface{} {
	return AsBigDecimal(x).AddP(AsBigDecimal(y))
}
func (o bigDecimalOps) Sub(x, y interface{}) interface{} {
	return AsBigDecimal(x).Sub(AsBigDecimal(y))
}
func (o bigDecimalOps) SubP(x, y interface{}) interface{} {
	return AsBigDecimal(x).SubP(AsBigDecimal(y))
}
func (o bigDecimalOps) Multiply(x, y interface{}) interface{} {
	return AsBigDecimal(x).Multiply(AsBigDecimal(y))
}
func (o bigDecimalOps) Divide(x, y interface{}) interface{} {
	return AsBigDecimal(x).Divide(AsBigDecimal(y))
}
func (o bigDecimalOps) LT(x, y interface{}) bool {
	return AsBigDecimal(x).LT(AsBigDecimal(y))
}
func (o bigDecimalOps) Max(x, y interface{}) interface{} {
	xx := AsBigDecimal(x)
	yy := AsBigDecimal(y)
	if xx.Cmp(yy) > 0 {
		return x
	}
	return y

}
func (o bigDecimalOps) Min(x, y interface{}) interface{} {
	xx := AsBigDecimal(x)
	yy := AsBigDecimal(y)
	if xx.Cmp(yy) < 0 {
		return x
	}
	return y

}
func (o bigDecimalOps) Equal(x, y interface{}) bool {
	return AsBigDecimal(x).Cmp(AsBigDecimal(y)) == 0
}

func (o float64Ops) IsPos(x interface{}) bool {
	return AsFloat64(x) > 0
}

func (o float64Ops) IsZero(x interface{}) bool {
	return AsFloat64(x) == 0
}

func (o float64Ops) Add(x, y interface{}) interface{} {
	return AsFloat64(x) + AsFloat64(y)
}
func (o float64Ops) AddP(x, y interface{}) interface{} {
	return AsFloat64(x) + AsFloat64(y)
}
func (o float64Ops) Sub(x, y interface{}) interface{} {
	return AsFloat64(x) - AsFloat64(y)
}
func (o float64Ops) SubP(x, y interface{}) interface{} {
	return AsFloat64(x) - AsFloat64(y)
}
func (o float64Ops) Multiply(x, y interface{}) interface{} {
	return AsFloat64(x) * AsFloat64(y)
}
func (o float64Ops) Divide(x, y interface{}) interface{} {
	return AsFloat64(x) / AsFloat64(y)
}
func (o float64Ops) LT(x, y interface{}) bool {
	return AsFloat64(x) < AsFloat64(y)
}
func (o float64Ops) Max(x, y interface{}) interface{} {
	if AsFloat64(x) > AsFloat64(y) {
		return x
	}
	return y

}
func (o float64Ops) Min(x, y interface{}) interface{} {
	if AsFloat64(x) < AsFloat64(y) {
		return x
	}
	return y

}
func (o float64Ops) Equal(x, y interface{}) bool {
	return AsFloat64(x) == AsFloat64(y)
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
func AsInt64(x interface{}) int64 {
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

func AsBigInt(x interface{}) *BigInt {
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
	default:
		panic("cannot convert to BigInt")
	}
}

func AsRatio(x interface{}) *Ratio {
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
	default:
		panic("cannot convert to Ratio")
	}
}

func AsBigDecimal(x interface{}) *BigDecimal {
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
	default:
		panic("cannot convert to BigDecimal")
	}
}

func AsFloat64(x interface{}) float64 {
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
	default:
		panic("cannot convert to float64")
	}
}
