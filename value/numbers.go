package value

import (
	"fmt"
	"math/big"
)

// BigDec is an arbitrary-precision decimal number. It wraps and has
// the same semantics as big.Float. big.Float is not used directly
// because it is mutable, and the core BigDecimal should not be.
type BigDecimal struct {
	val *big.Float
}

// NewBigDecimal creates a new BigDecimal from a string.
func NewBigDecimal(s string) (*BigDecimal, error) {
	bf, ok := new(big.Float).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid big decimal: %s", s)
	}
	return &BigDecimal{val: bf}, nil
}

// NewBigDecimalFromFloat64 creates a new BigDecimal from a float64.
func NewBigDecimalFromFloat64(x float64) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetFloat64(x)}
}

func (n *BigDecimal) String() string {
	return n.val.String() + "M"
}

func (n *BigDecimal) Equal(v interface{}) bool {
	other, ok := v.(*BigDecimal)
	if !ok {
		return false
	}
	return n.val.Cmp(other.val) == 0
}

func (n *BigDecimal) AddInt(x int) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Add(n.val, big.NewFloat(float64(x)))}
}

func (n *BigDecimal) Add(other *BigDecimal) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Add(n.val, other.val)}
}

func (n *BigDecimal) AddP(other *BigDecimal) *BigDecimal {
	return n.Add(other)
}

func (n *BigDecimal) Sub(other *BigDecimal) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Sub(n.val, other.val)}
}

func (n *BigDecimal) SubP(other *BigDecimal) *BigDecimal {
	return n.Sub(other)
}

func (n *BigDecimal) Cmp(other *BigDecimal) int {
	return n.val.Cmp(other.val)
}

func (n *BigDecimal) LT(other *BigDecimal) bool {
	return n.Cmp(other) < 0
}

// BigInt is an arbitrary-precision integer. It wraps and has the same
// semantics as big.Int. big.Int is not used directly because it is
// mutable, and the core BigInt should not be.
type BigInt struct {
	val *big.Int
}

// NewBigInt creates a new BigInt from a string.
func NewBigInt(s string) (*BigInt, error) {
	bi, ok := new(big.Int).SetString(s, 0)
	if !ok {
		return nil, fmt.Errorf("invalid big int: %s", s)
	}
	return &BigInt{val: bi}, nil
}

// NewBigIntFromInt64 creates a new BigInt from an int64.
func NewBigIntFromInt64(x int64) *BigInt {
	return &BigInt{val: big.NewInt(x)}
}

func (n *BigInt) String() string {
	return n.val.String() + "N"
}

func (n *BigInt) Equal(v interface{}) bool {
	other, ok := v.(*BigInt)
	if !ok {
		return false
	}
	return n.val.Cmp(other.val) == 0
}

func (n *BigInt) AddInt(x int) *BigInt {
	return &BigInt{val: new(big.Int).Add(n.val, big.NewInt(int64(x)))}
}

func (n *BigInt) Add(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Add(n.val, other.val)}
}

func (n *BigInt) AddP(other *BigInt) *BigInt {
	return n.Add(other)
}

func (n *BigInt) Sub(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Sub(n.val, other.val)}
}

func (n *BigInt) SubP(other *BigInt) *BigInt {
	return n.Sub(other)
}

func (n *BigInt) Cmp(other *BigInt) int {
	return n.val.Cmp(other.val)
}

func (n *BigInt) LT(other *BigInt) bool {
	return n.Cmp(other) < 0
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

// AsInt64 returns any integral value as an int64. If the value cannot
// be represented as an int64, it returns false. Floats are not
// converted.
// func AsInt64(v interface{}) (int64, bool) {
// 	switch v := v.(type) {
// 	case int:
// 		return int64(v), true
// 	case int64:
// 		return v, true
// 	case int32:
// 		return int64(v), true
// 	case int16:
// 		return int64(v), true
// 	case int8:
// 		return int64(v), true
// 	case uint:
// 		return int64(v), true
// 	case uint64:
// 		return int64(v), true
// 	case uint32:
// 		return int64(v), true
// 	case uint16:
// 		return int64(v), true
// 	case uint8:
// 		return int64(v), true
// 	default:
// 		return 0, false
// 	}
// }

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
