package lang

import (
	"math/big"
)

// Ratio is a value that represents a ratio.
type Ratio struct {
	val *big.Rat
}

// NewRatio creates a new ratio value.
func NewRatio(numerator, denominator int64) *Ratio {
	return &Ratio{
		val: big.NewRat(numerator, denominator),
	}
}

func NewRatioBigInt(num, den *BigInt) *Ratio {
	return &Ratio{
		val: new(big.Rat).SetFrac(num.val, den.val),
	}
}

func NewRatioGoBigInt(num, den *big.Int) *Ratio {
	return &Ratio{
		val: new(big.Rat).SetFrac(num, den),
	}
}

func (r *Ratio) Numerator() *big.Int {
	return new(big.Int).Set(r.val.Num())
}

func (r *Ratio) Denominator() *big.Int {
	return new(big.Int).Set(r.val.Denom())
}

func (r *Ratio) BigIntegerValue() *big.Int {
	return new(big.Int).Div(r.val.Num(), r.val.Denom())
}

func (r *Ratio) String() string {
	return r.val.RatString()
}

func (r *Ratio) Equals(other interface{}) bool {
	if other, ok := other.(*Ratio); ok {
		return r.val.Cmp(other.val) == 0
	}
	return false
}

func (r *Ratio) Add(other *Ratio) *Ratio {
	return &Ratio{
		val: new(big.Rat).Add(r.val, other.val),
	}
}

func (r *Ratio) AddP(other *Ratio) *Ratio {
	return r.Add(other)
}

func (r *Ratio) Sub(other *Ratio) *Ratio {
	return &Ratio{
		val: new(big.Rat).Sub(r.val, other.val),
	}
}

func (r *Ratio) SubP(other *Ratio) *Ratio {
	return r.Sub(other)
}

func (r *Ratio) Multiply(other *Ratio) any {
	xn, xd := r.Numerator(), r.Denominator()
	yn, yd := other.Numerator(), other.Denominator()
	return Divide(
		xn.Mul(xn, yn),
		xd.Mul(xd, yd))
}

func (r *Ratio) Divide(other *Ratio) any {
	xn, xd := r.Numerator(), r.Denominator()
	yn, yd := other.Numerator(), other.Denominator()
	return Divide(
		xn.Mul(xn, yd),
		xd.Mul(xd, yn))
}

func (r *Ratio) Quotient(other *Ratio) any {
	xn, xd := r.Numerator(), r.Denominator()
	yn, yd := other.Numerator(), other.Denominator()

	qn := new(big.Int).Mul(xn, yd)
	q := qn.Div(qn, xd.Mul(xd, yn))
	return NewBigIntFromGoBigInt(q)
}

func (r *Ratio) Cmp(other *Ratio) int {
	return r.val.Cmp(other.val)
}

func (r *Ratio) LT(other *Ratio) bool {
	return r.Cmp(other) < 0
}

func (r *Ratio) LTE(other *Ratio) bool {
	return r.Cmp(other) <= 0
}

func (r *Ratio) GT(other *Ratio) bool {
	return r.Cmp(other) > 0
}

func (r *Ratio) GTE(other *Ratio) bool {
	return r.Cmp(other) >= 0
}

func (r *Ratio) Abs() *Ratio {
	if r.val.Sign() < 0 {
		return &Ratio{
			val: new(big.Rat).Abs(r.val),
		}
	}
	return r
}
