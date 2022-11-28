package value

import (
	"math/big"
	"strconv"
)

// NewNum returns a new number value. This function is deprecated. Use
// floats directly.
func NewNum(n float64) float64 {
	return n
}

// Long is a 64-bit integer.
type Long struct {
	Section
	Value int64
}

func NewLong(n int64, opts ...Option) *Long {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Long{
		Section: o.section,
		Value:   n,
	}
}

func (n *Long) String() string {
	return strconv.FormatInt(n.Value, 10)
}

func (n *Long) Equal(v interface{}) bool {
	other, ok := v.(*Long)
	if !ok {
		return false
	}
	return n.Value == other.Value
}

func (n *Long) GoValue() interface{} {
	return n.Value
}

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
