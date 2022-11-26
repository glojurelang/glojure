package value

import "strconv"

// Num is a number.
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
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

func (n *Num) Equal(v Value) bool {
	other, ok := v.(*Num)
	if !ok {
		return false
	}
	return n.Value == other.Value
}

func (n *Num) GoValue() interface{} {
	return n.Value
}
