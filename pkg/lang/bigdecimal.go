package lang

import (
	"fmt"
	"math/big"

	"bitbucket.org/pcastools/hash"
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

// NewBigDecimalFromBigFloat
func NewBigDecimalFromBigFloat(x *big.Float) *BigDecimal {
	xCopy := new(big.Float)
	xCopy.Set(x)
	return &BigDecimal{val: xCopy}
}

// NewBigDecimalFromFloat64 creates a new BigDecimal from a float64.
func NewBigDecimalFromFloat64(x float64) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetFloat64(x)}
}

func NewBigDecimalFromInt64(x int64) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetInt64(x)}
}

func NewBigDecimalFromRatio(x *Ratio) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetRat(x.val)}
}

func (n *BigDecimal) ToBigInteger() *big.Int {
	res, _ := n.val.Int(nil)
	return res
}

func (n *BigDecimal) ToBigFloat() *big.Float {
	res := new(big.Float)
	res.Set(n.val)
	return res
}

func (n *BigDecimal) String() string {
	return n.val.String()
}

func (n *BigDecimal) Hash() uint32 {
	if n.val.Sign() == 0 {
		return 0
	}
	return hash.String(n.val.String())
}

func (n *BigDecimal) Equals(v interface{}) bool {
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

func (n *BigDecimal) Multiply(other *BigDecimal) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Mul(n.val, other.val)}
}

func (n *BigDecimal) Divide(other *BigDecimal) *BigDecimal {
	// Todo: div
	return &BigDecimal{val: new(big.Float).Quo(n.val, other.val)}
}

func (n *BigDecimal) Quotient(other *BigDecimal) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Quo(n.val, other.val)}
}

func (n *BigDecimal) Remainder(other *BigDecimal) *BigDecimal {
	panic("not implemented")
}

func (n *BigDecimal) Cmp(other *BigDecimal) int {
	return n.val.Cmp(other.val)
}

func (n *BigDecimal) LT(other *BigDecimal) bool {
	return n.Cmp(other) < 0
}

func (n *BigDecimal) LTE(other *BigDecimal) bool {
	return n.Cmp(other) <= 0
}

func (n *BigDecimal) GT(other *BigDecimal) bool {
	return n.Cmp(other) > 0
}

func (n *BigDecimal) GTE(other *BigDecimal) bool {
	return n.Cmp(other) >= 0
}

func (n *BigDecimal) Negate() *BigDecimal {
	return &BigDecimal{val: new(big.Float).Neg(n.val)}
}

func (n *BigDecimal) Abs() *BigDecimal {
	if n.val.Sign() < 0 {
		return &BigDecimal{val: new(big.Float).Abs(n.val)}
	}
	return n
}
