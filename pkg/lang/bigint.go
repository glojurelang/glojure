package lang

import (
	"fmt"
	"math/big"
)

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

func NewBigIntFromGoBigInt(x *big.Int) *BigInt {
	xCopy := new(big.Int)
	xCopy.Set(x)
	return &BigInt{val: xCopy}
}

func (n *BigInt) String() string {
	return n.val.String()
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

func (n *BigInt) Multiply(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Mul(n.val, other.val)}
}

func (n *BigInt) Divide(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Div(n.val, other.val)}
}

func (n *BigInt) Remainder(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Rem(n.val, other.val)}
}

func (n *BigInt) Cmp(other *BigInt) int {
	return n.val.Cmp(other.val)
}

func (n *BigInt) LT(other *BigInt) bool {
	return n.Cmp(other) < 0
}

func (n *BigInt) LTE(other *BigInt) bool {
	return n.Cmp(other) <= 0
}

func (n *BigInt) GT(other *BigInt) bool {
	return n.Cmp(other) > 0
}

func (n *BigInt) GTE(other *BigInt) bool {
	return n.Cmp(other) >= 0
}
