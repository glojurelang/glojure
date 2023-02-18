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

func (n *BigDecimal) Divide(other *BigDecimal) *BigDecimal {
	return &BigDecimal{val: new(big.Float).Quo(n.val, other.val)}
}

func (n *BigDecimal) Cmp(other *BigDecimal) int {
	return n.val.Cmp(other.val)
}

func (n *BigDecimal) LT(other *BigDecimal) bool {
	return n.Cmp(other) < 0
}
