package value

import (
	"fmt"
	"math/big"
	"strconv"
)

// Num is a floating point number.
type Num struct {
	Section
	// Value is the value of the number. It should not be modified
	// unless the number is being used transiently, because language
	// semantics require that values are immutable.
	Value float64
}

func NewNum(n float64, opts ...Option) *Num {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Num{
		Section: o.section,
		Value:   n,
	}
}

func (n *Num) String() string {
	if n.Value == float64(int64(n.Value)) {
		return fmt.Sprintf("%d.0", int64(n.Value))
	}
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

func (n *Num) Equal(v interface{}) bool {
	other, ok := v.(*Num)
	if !ok {
		return false
	}
	return n.Value == other.Value
}

func (n *Num) GoValue() interface{} {
	return n.Value
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
