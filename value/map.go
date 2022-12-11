package value

import (
	"fmt"
	"strings"
)

type (
	// Map represents a map of glojure values.
	Map struct {
		Section
		keyVals []interface{}
	}

	// MapEntry represents a key-value pair in a map.
	MapEntry struct {
		Key, Value interface{}
	}

	MapSeq struct {
		m          *Map
		entryIndex int
	}
)

var (
	_ IPersistentMap = (*Map)(nil)
)

func NewMapSeq(m *Map) ISeq {
	if m.Count() == 0 {
		return emptyList
	}
	return &MapSeq{
		m:          m,
		entryIndex: 0,
	}
}

func (s *MapSeq) First() interface{} {
	return MapEntry{
		Key:   s.m.keyVals[2*s.entryIndex],
		Value: s.m.keyVals[2*s.entryIndex+1],
	}
}

func (s *MapSeq) Next() ISeq {
	if s.entryIndex >= s.m.Count() {
		return nil
	}
	return &MapSeq{
		m:          s.m,
		entryIndex: s.entryIndex + 1,
	}
}

func (s *MapSeq) Rest() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (s *MapSeq) IsEmpty() bool {
	return s.entryIndex >= s.m.Count()
}

func NewMap(keyVals []interface{}, opts ...Option) IPersistentMap {
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

func (m *Map) EntryAt(k interface{}) (interface{}, bool) {
	return m.ValueAt(k)
}

func (m *Map) ContainsKey(key interface{}) bool {
	_, ok := m.ValueAt(key)
	return ok
}

func (m *Map) Conj(x interface{}) Conjer {
	switch x := x.(type) {
	case MapEntry:
		return m.Assoc(x.Key, x.Value).(Conjer)
	case IPersistentVector:
		if x.Count() != 2 {
			panic("vector arg to map conj must be a pair")
		}
		return m.Assoc(MustNth(x, 0), MustNth(x, 1)).(Conjer)
	}

	var ret Conjer = m
	seq := Seq(x)
	for !seq.IsEmpty() {
		ret = ret.Conj(seq.First().(MapEntry))
		seq = seq.Rest()
	}
	return ret
}

func (m *Map) Assoc(k, v interface{}) Associative {
	return NewMap(append(m.keyVals, k, v))
}

func (m *Map) AssocEx(k, v interface{}) (IPersistentMap, error) {
	if _, ok := m.ValueAt(k); ok {
		return nil, fmt.Errorf("key %v already exists", k)
	}
	return m.Assoc(k, v).(IPersistentMap), nil
}

func (m *Map) Without(k interface{}) IPersistentMap {
	newKeyVals := make([]interface{}, 0, len(m.keyVals))
	for i := 0; i < len(m.keyVals); i += 2 {
		if !Equal(m.keyVals[i], k) {
			newKeyVals = append(newKeyVals, m.keyVals[i], m.keyVals[i+1])
		}
	}
	return NewMap(newKeyVals)
}

func (m *Map) Count() int {
	return len(m.keyVals) / 2
}

func (m *Map) Seq() ISeq {
	return NewMapSeq(m)
}

// func (m *Map) First() interface{} {
// 	if m.Count() == 0 {
// 		return nil
// 	}

// 	return NewVector([]interface{}{m.keyVals[0], m.keyVals[1]})
// }

// func (m *Map) Rest() ISeq {
// 	if m.Count() == 0 {
// 		return emptyList
// 	}

// 	return NewMap(m.keyVals[2:])
// }

func (m *Map) IsEmpty() bool {
	return m.Count() == 0
}

func (m *Map) String() string {
	b := strings.Builder{}

	first := true

	// TODO: factor out common namespace
	b.WriteString("{")
	for i := 0; i < len(m.keyVals); i += 2 {
		if !first {
			b.WriteString(", ")
		}
		first = false

		k, v := m.keyVals[i], m.keyVals[i+1]

		b.WriteString(ToString(k))
		b.WriteRune(' ')
		b.WriteString(ToString(v))
	}
	b.WriteString("}")
	return b.String()
}

func (m *Map) Equal(v2 interface{}) bool {
	// TODO: implement me
	return false
}
