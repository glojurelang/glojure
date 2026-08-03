package lang

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"bitbucket.org/pcastools/hash"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// BigDec is an arbitrary-precision floating point number. It wraps
// and has the same semantics as big.Float. big.Float is not used
// directly because it is mutable, and the core BigDecimal should not
// be.
//
// TODO: swap out with a *decimal* representation. The go standard
// library big.Float is a binary floating point representation,
// which means that some decimal fractions cannot be represented
// exactly. This can lead to unexpected results when doing
// arithmetic with decimal fractions. A decimal representation
// would avoid this problem.
type BigDecimal struct {
	val      *big.Float
	scale    int
	hasScale bool
}

type RoundingMode string

const (
	RoundingModeDown     RoundingMode = "DOWN"
	RoundingModeHalfEven RoundingMode = "HALF_EVEN"
)

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

func NewBigDecimalFromBigInt(x *big.Int) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetInt(x)}
}

func NewBigDecimalFromRatio(x *Ratio) *BigDecimal {
	return &BigDecimal{val: new(big.Float).SetRat(x.val)}
}

func (n *BigDecimal) ToBigInteger() *big.Int {
	res, _ := n.val.Int(nil)
	return res
}

// SetScale implements the java.math.BigDecimal rounding modes used by
// portable Clojure libraries. DOWN truncates toward zero and HALF_EVEN uses
// banker's rounding.
func (n *BigDecimal) SetScale(scale any, mode any) *BigDecimal {
	digits := MustAsInt(scale)
	if digits < 0 {
		panic(fmt.Errorf("BigDecimal.setScale: negative scale %d is unsupported", digits))
	}

	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	rat, _ := n.val.Rat(nil)
	scaledNumerator := new(big.Int).Mul(rat.Num(), factor)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaledNumerator, rat.Denom(), remainder)

	if fmt.Sprint(mode) == string(RoundingModeHalfEven) && remainder.Sign() != 0 {
		twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(remainder), 1)
		denominator := new(big.Int).Abs(rat.Denom())
		comparison := twiceRemainder.Cmp(denominator)
		if comparison > 0 || (comparison == 0 && quotient.Bit(0) == 1) {
			if scaledNumerator.Sign() < 0 {
				quotient.Sub(quotient, big.NewInt(1))
			} else {
				quotient.Add(quotient, big.NewInt(1))
			}
		}
	}

	result := new(big.Rat).SetFrac(quotient, factor)
	return &BigDecimal{
		val:      new(big.Float).SetPrec(256).SetRat(result),
		scale:    digits,
		hasScale: true,
	}
}

func (n *BigDecimal) ToPlainString() string {
	if n.hasScale {
		return n.val.Text('f', n.scale)
	}
	return n.val.Text('f', -1)
}

func (n *BigDecimal) ToBigFloat() *big.Float {
	res := new(big.Float)
	res.Set(n.val)
	return res
}

func (n *BigDecimal) String() string {
	s := n.val.Text('f', -1)
	// Ensure decimal point is present (e.g. "0" → "0.0")
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// StripTrailingZeros returns a string representation with trailing
// fractional zeros removed (e.g. "1.0" → "1", "1.50" → "1.5").
func (n *BigDecimal) StripTrailingZeros() string {
	s := n.val.Text('f', -1)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
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
	// Truncate toward zero (integer quotient)
	quo := new(big.Float).Quo(n.val, other.val)
	intQuo, _ := quo.Int(nil)
	return &BigDecimal{val: new(big.Float).SetInt(intQuo)}
}

func (n *BigDecimal) Remainder(other *BigDecimal) *BigDecimal {
	quotient := new(big.Float).Quo(n.val, other.val)
	intQuotient, _ := quotient.Int(nil)
	intQuotientFloat := new(big.Float).SetInt(intQuotient)
	product := new(big.Float).Mul(intQuotientFloat, other.val)
	remainder := new(big.Float).Sub(n.val, product)
	return &BigDecimal{val: remainder}
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

func newHostBigDecimal(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Errorf("BigDecimal/new: wrong number of args (%d)", len(args)))
	}
	if text, ok := args[0].(string); ok {
		value, err := NewBigDecimal(text)
		if err != nil {
			panic(err)
		}
		return value
	}
	return AsBigDecimal(args[0])
}

func init() {
	pkgmap.SetHostClassPackage("BigDecimal", "java.math")
	pkgmap.SetHostClass("BigDecimal",
		NewClass(reflect.TypeOf((*BigDecimal)(nil)), "java.math.BigDecimal"))
	RegisterHostConstructor("java.math.BigDecimal",
		FnFunc(func(args ...any) any { return newHostBigDecimal(args...) }))

	pkgmap.SetHostClassPackage("RoundingMode", "java.math")
	pkgmap.SetHostClass("RoundingMode",
		NewClass(reflect.TypeOf(RoundingMode("")), "java.math.RoundingMode"))
	for _, prefix := range []string{"RoundingMode", "java.math.RoundingMode"} {
		pkgmap.Set(prefix+".DOWN", RoundingModeDown)
		pkgmap.Set(prefix+".HALF_EVEN", RoundingModeHalfEven)
	}
}
