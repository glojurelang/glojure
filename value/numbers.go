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

// NewBigDecimal creates a new BigDecimal from a string
func NewBigDecimal(s string) (*BigDecimal, error) {
	bf, ok := new(big.Float).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid big decimal: %s", s)
	}
	return &BigDecimal{val: bf}, nil
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
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	default:
		return 0, false
	}
}
