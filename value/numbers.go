package value

import (
	"math/big"
)

// BigDec is an arbitrary-precision decimal number. It wraps and has
// the same semantics as big.Float. big.Float is not used directly
// because it is mutable, and the core BigDecimal should not be.
type BigDecimal struct {
	val big.Float
}

func NewBigDecimal(n big.Float) *BigDecimal {
	return &BigDecimal{val: n}
}

func (n *BigDecimal) String() string {
	return n.val.String() + "M"
}

func (n *BigDecimal) Equal(v interface{}) bool {
	other, ok := v.(*BigDecimal)
	if !ok {
		return false
	}
	return n.val.Cmp(&other.val) == 0
}

func (n *BigDecimal) Pos() Pos {
	return Pos{}
}

func (n *BigDecimal) End() Pos {
	return Pos{}
}

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
