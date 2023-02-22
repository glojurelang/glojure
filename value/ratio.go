package value

import "math/big"

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

func (r *Ratio) String() string {
	return r.val.RatString()
}

func (r *Ratio) Equal(other interface{}) bool {
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

func (r *Ratio) Multiply(other *Ratio) *Ratio {
	return &Ratio{
		val: new(big.Rat).Mul(r.val, other.val),
	}
}

func (r *Ratio) Divide(other *Ratio) *Ratio {
	return &Ratio{
		val: new(big.Rat).Quo(r.val, other.val),
	}
}

func (r *Ratio) Cmp(other *Ratio) int {
	return r.val.Cmp(other.val)
}

func (r *Ratio) LT(other *Ratio) bool {
	return r.Cmp(other) < 0
}

func (r *Ratio) GT(other *Ratio) bool {
	return r.Cmp(other) > 0
}
