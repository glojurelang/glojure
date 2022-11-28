package value

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/persistent/vector"
)

// Vector is a vector of values.
type Vector struct {
	Section
	vec vector.Vector
}

func NewVector(values []Value, opts ...Option) *Vector {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	vals := make([]interface{}, len(values))
	for i, v := range values {
		vals[i] = v
	}
	vec := vector.New(vals...)

	return &Vector{
		Section: o.section,
		vec:     vec,
	}
}

func (v *Vector) Count() int {
	return v.vec.Len()
}

func (v *Vector) Conj(items ...Value) Conjer {
	vec := v.vec
	for _, item := range items {
		vec = vec.Conj(item)
	}
	return &Vector{vec: vec}
}

func (v *Vector) ValueAt(i int) Value {
	val, ok := v.vec.Index(i)
	if !ok {
		panic("index out of range")
	}
	if val == nil {
		return nil
	}
	return val.(Value)
}

func (v *Vector) Nth(i int) (val Value, ok bool) {
	if i < 0 || i >= v.Count() {
		return nil, false
	}
	return v.ValueAt(i), true
}

func (v *Vector) SubVector(start, end int) *Vector {
	return &Vector{vec: v.vec.SubVector(start, end)}
}

func (v *Vector) Enumerate() (<-chan Value, func()) {
	rest := v.vec
	return enumerateFunc(func() (Value, bool) {
		if rest.Len() == 0 {
			return nil, false
		}
		val, _ := rest.Index(0)
		rest = rest.SubVector(1, rest.Len())
		return val.(Value), true
	})
}

func (v *Vector) String() string {
	b := strings.Builder{}

	b.WriteString("[")
	for i := 0; i < v.Count(); i++ {
		el := v.ValueAt(i)
		if el == nil {
			b.WriteString("nil")
		} else {
			b.WriteString(el.String())
		}
		if i < v.Count()-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("]")
	return b.String()
}

func (v *Vector) Equal(v2 interface{}) bool {
	other, ok := v2.(*Vector)
	if !ok {
		return false
	}
	if v.Count() != other.Count() {
		return false
	}
	for i := 0; i < v.Count(); i++ {
		vVal, oVal := v.ValueAt(i), other.ValueAt(i)
		if vVal == nil || oVal == nil {
			return vVal == oVal
		}
		if !Equal(vVal, oVal) {
			return false
		}
	}
	return true
}

func (v *Vector) Apply(env Environment, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector apply expects 1 argument, got %d", len(args))
	}

	index, ok := args[0].(*Long)
	if !ok {
		return nil, fmt.Errorf("vector apply takes an int as an argument")
	}

	i := int(index.Value)
	if i < 0 || i >= v.Count() {
		return nil, fmt.Errorf("index out of bounds")
	}

	return v.ValueAt(i), nil
}

func (v *Vector) GoValue() interface{} {
	var vals []interface{}
	for i := 0; i < v.Count(); i++ {
		val := v.ValueAt(i)
		if val == nil {
			vals = append(vals, nil)
			continue
		}

		if gv, ok := val.(GoValuer); ok {
			vals = append(vals, gv.GoValue())
			continue
		}

		vals = append(vals, val)
	}
	return vals
}
