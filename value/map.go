package value

import (
	"strings"
)

// Map represents a map of glojure values.
type Map struct {
	Section
	keyVals []Value
}

func NewMap(keyVals []Value, opts ...Option) *Map {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	return &Map{
		Section: o.section,
		keyVals: keyVals,
	}
}

func (m *Map) Count() int {
	return len(m.keyVals) / 2
}

// func (v *Vector) Conj(items ...Value) Conjer {
// 	vec := v.vec
// 	for _, item := range items {
// 		vec = vec.Conj(item)
// 	}
// 	return &Vector{vec: vec}
// }

func (m *Map) First() Value {
	if m.Count() == 0 {
		return NilValue
	}

	return NewVector([]Value{m.keyVals[0], m.keyVals[1]})
}

func (m *Map) Rest() Sequence {
	if m.Count() == 0 {
		return NilValue
	}

	return NewMap(m.keyVals[2:])
}

func (m *Map) IsEmpty() bool {
	return m.Count() == 0
}

func (m *Map) String() string {
	b := strings.Builder{}

	first := true

	b.WriteString("{")
	for ; !m.IsEmpty(); m = m.Rest().(*Map) {
		if !first {
			b.WriteString(", ")
		}
		first = false

		el := m.First().(*Vector)

		b.WriteString(el.ValueAt(0).String())
		b.WriteRune(' ')
		b.WriteString(el.ValueAt(1).String())
	}
	b.WriteString("}")
	return b.String()
}

func (m *Map) Equal(v2 interface{}) bool {
	// if v == v2 {
	// 	return true
	// }
	// other, ok := v2.(*Map)
	// if !ok {
	// 	return false
	// }
	return false
}

// func (v *Vector) Apply(env Environment, args []Value) (Value, error) {
// }

// func (v *Vector) GoValue() interface{} {
// 	var vals []interface{}
// 	for i := 0; i < v.Count(); i++ {
// 		val := v.ValueAt(i)
// 		if val == nil {
// 			vals = append(vals, nil)
// 			continue
// 		}

// 		if gv, ok := val.(GoValuer); ok {
// 			vals = append(vals, gv.GoValue())
// 			continue
// 		}

// 		vals = append(vals, val)
// 	}
// 	return vals
// }
