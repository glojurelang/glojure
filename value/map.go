package value

import (
	"strings"
)

// Map represents a map of glojure values.
type Map struct {
	Section
	keyVals []interface{}
}

func NewMap(keyVals []interface{}, opts ...Option) *Map {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if len(keyVals)%2 != 0 {
		panic("invalid map. must have even number of inputs")
	}

	return &Map{
		Section: o.section,
		keyVals: keyVals,
	}
}

func (m *Map) ValueAt(key interface{}) (interface{}, bool) {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equal(m.keyVals[i], key) {
			return m.keyVals[i+1], true
		}
	}

	return nil, false
}

// TODO: define and amend to implement and return the right interface
func (m *Map) Assoc(k, v interface{}) *Map {
	return NewMap(append(m.keyVals, k, v))
}

func (m *Map) Count() int {
	return len(m.keyVals) / 2
}

func (m *Map) First() interface{} {
	if m.Count() == 0 {
		return nil
	}

	return NewVector([]interface{}{m.keyVals[0], m.keyVals[1]})
}

func (m *Map) Rest() ISeq {
	if m.Count() == 0 {
		return emptyList
	}

	return NewMap(m.keyVals[2:])
}

func (m *Map) IsEmpty() bool {
	return m.Count() == 0
}

func (m *Map) String() string {
	b := strings.Builder{}

	first := true

	// TODO: factor out common namespace
	b.WriteString("{")
	for ; !m.IsEmpty(); m = m.Rest().(*Map) {
		if !first {
			b.WriteString(", ")
		}
		first = false

		el := m.First().(*Vector)

		b.WriteString(ToString(el.ValueAt(0)))
		b.WriteRune(' ')
		b.WriteString(ToString(el.ValueAt(1)))
	}
	b.WriteString("}")
	return b.String()
}

func (m *Map) Equal(v2 interface{}) bool {
	// TODO: implement me
	return false
}
