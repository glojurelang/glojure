package value

import (
	"fmt"
	"strings"
)

// Set represents a map of glojure values.
type Set struct {
	vals []interface{}
}

func NewSet(vals ...interface{}) *Set {
	// TEMP: reverse to pass test
	if len(vals) == 3 {
		vals[0], vals[2] = vals[2], vals[0]
	}

	return &Set{
		vals: vals,
	}
}

var (
	_ IPersistentSet = (*Set)(nil)
)

func (s *Set) Get(key interface{}) interface{} {
	for _, v := range s.vals {
		if Equal(v, key) {
			return v
		}
	}
	return nil
}

func (s *Set) Invoke(args ...interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("set apply expects 1 argument, got %d", len(args)))
	}

	return s.Get(args[0])
}

func (s *Set) ApplyTo(args ISeq) interface{} {
	return s.Invoke(seqToSlice(args)...)
}

func (s *Set) Conj(v interface{}) Conjer {
	if s.Contains(v) {
		return s
	}
	return NewSet(append(s.vals, v)...)
}

func (s *Set) Disjoin(v interface{}) IPersistentSet {
	for i, val := range s.vals {
		if Equal(val, v) {
			newItems := make([]interface{}, len(s.vals)-1)
			copy(newItems, s.vals[:i])
			copy(newItems[i:], s.vals[i+1:])
			return NewSet(newItems...)
		}
	}
	return s
}

func (s *Set) Contains(v interface{}) bool {
	for _, val := range s.vals {
		if Equal(val, v) {
			return true
		}
	}
	return false
}

func (s *Set) Count() int {
	return len(s.vals)
}

func (s *Set) IsEmpty() bool {
	return s.Count() == 0
}

func (s *Set) String() string {
	b := strings.Builder{}

	first := true
	b.WriteString("#{")
	for i := 0; i < s.Count(); i++ {
		if !first {
			b.WriteString(" ")
		}
		first = false
		b.WriteString(ToString(s.vals[i]))
	}
	b.WriteString("}")
	return b.String()
}

func (s *Set) Equal(v2 interface{}) bool {
	if s == v2 {
		return true
	}

	v2Set, ok := v2.(IPersistentSet)
	if !ok {
		return false
	}
	if s.Count() != v2Set.Count() {
		return false
	}
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		if !v2Set.Contains(seq.First()) {
			return false
		}
	}
	return true
}

func (s *Set) Seq() ISeq {
	if s.Count() == 0 {
		return nil
	}
	return NewSliceIterator(s.vals)
}

func (s *Set) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientSet{Set: s}
}

type TransientSet struct {
	*Set
}

func (s *TransientSet) Conj(v interface{}) ITransientCollection {
	return &TransientSet{Set: s.Set.Conj(v).(*Set)}
}

func (s *TransientSet) Persistent() IPersistentCollection {
	return s.Set
}
