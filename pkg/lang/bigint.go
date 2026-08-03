package lang

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"bitbucket.org/pcastools/hash"
)

// BigInt is an arbitrary-precision integer. It wraps and has the same
// semantics as big.Int. big.Int is not used directly because it is
// mutable, and the core BigInt should not be.
type BigInt struct {
	val *big.Int
}

// BigIntStringFromFloat64 converts a float64 to its decimal string
// representation with the fractional part truncated, suitable for
// parsing as a BigInt. This preserves the exact decimal digits rather
// than going through binary float representation.
func BigIntStringFromFloat64(x float64) string {
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if i := strings.Index(s, "."); i != -1 {
		s = s[:i]
	}
	return s
}

// NewBigInt creates a new BigInt from a string.
func NewBigInt(s string) (*BigInt, error) {
	return NewBigIntWithBase(s, 0)
}

// NewBigIntWithBase creates a new BigInt from a string.
func NewBigIntWithBase(s string, base int) (*BigInt, error) {
	bi, ok := new(big.Int).SetString(s, base)
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

func (n *BigInt) Int64() int64 {
	return n.val.Int64()
}

func (n *BigInt) IsInt64() bool {
	return n.val.IsInt64()
}

func (n *BigInt) ToBigInteger() *big.Int {
	return new(big.Int).Set(n.val)
}

func (n *BigInt) ToBigDecimal() *BigDecimal {
	return NewBigDecimalFromBigFloat(new(big.Float).SetInt(n.val))
}

func (n *BigInt) String() string {
	return n.val.String()
}

// ToString mirrors java.math.BigInteger.toString, including its radix
// overload used by portable JVM libraries.
func (n *BigInt) ToString(args ...any) string {
	if len(args) == 0 {
		return n.String()
	}
	if len(args) != 1 {
		panic(fmt.Sprintf("BigInteger.toString: wrong number of args (%d)", len(args)))
	}
	return n.val.Text(MustAsInt(args[0]))
}

func (n *BigInt) ResolveFieldOrMethod(name string) (any, bool) {
	if strings.EqualFold(name, "toString") {
		return FnFunc(func(args ...any) any { return n.ToString(args...) }), true
	}
	return nil, false
}

func (n *BigInt) Hash() uint32 {
	if n.val.IsInt64() {
		return uint32(hash.Int64(n.val.Int64()))
	}
	return uint32(hash.ByteSlice(n.val.Bytes()))
}

func (n *BigInt) Equals(v interface{}) bool {
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

func (n *BigInt) Sub(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Sub(n.val, other.val)}
}

func (n *BigInt) Multiply(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Mul(n.val, other.val)}
}

var (
	bigIntZero   = big.NewInt(0)
	bigIntOne    = big.NewInt(1)
	bigIntNegOne = big.NewInt(-1)
)

func (n *BigInt) Divide(other *BigInt) any {
	if other.val.Sign() == 0 {
		panic(NewArithmeticError("divide by zero"))
	}
	gcd := new(big.Int).GCD(nil, nil, n.val, other.val)
	if gcd.Sign() == 0 {
		return &BigInt{val: bigIntZero}
	}
	num := new(big.Int).Div(n.val, gcd)
	den := new(big.Int).Div(other.val, gcd)
	// if d == 1, return n
	if den.Cmp(bigIntOne) == 0 {
		return &BigInt{val: num}
	}
	if den.Cmp(bigIntNegOne) == 0 {
		return &BigInt{val: num.Neg(num)}
	}
	return NewRatioGoBigInt(num, den)
}

func (n *BigInt) Quotient(other *BigInt) *BigInt {
	return &BigInt{val: new(big.Int).Quo(n.val, other.val)}
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

func (n *BigInt) Abs() *BigInt {
	if n.val.Sign() < 0 {
		return &BigInt{val: new(big.Int).Abs(n.val)}
	}
	return n
}
