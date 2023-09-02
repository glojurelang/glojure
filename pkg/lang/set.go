package lang

import (
	"fmt"
)

// Set represents a map of glojure values.
type Set struct {
	meta         IPersistentMap
	hash, hasheq uint32

	vals []interface{}
}

type PersistentHashSet = Set // hack until we have a proper persistent hash set

func CreatePersistentTreeSet(keys ISeq) interface{} {
	// TODO: implement
	return NewSet(seqToSlice(keys)...)
}

func CreatePersistentTreeSetWithComparator(comparator IFn, keys ISeq) interface{} {
	// TODO: implement
	return NewSet(seqToSlice(keys)...)
}

func NewSet(vals ...interface{}) *Set {
	// check for duplicates
	for i := 0; i < len(vals); i++ {
		for j := i + 1; j < len(vals); j++ {
			if Equiv(vals[i], vals[j]) {
				panic(NewIllegalArgumentError(fmt.Sprintf("duplicate key: %v", vals[i])))
			}
		}
	}

	return &Set{
		vals: vals,
	}
}

var (
	_ APersistentSet        = (*Set)(nil)
	_ IObj                  = (*Set)(nil)
	_ IPersistentCollection = (*Set)(nil)

	emptySet = NewSet()
)

func (s *Set) Get(key interface{}) interface{} {
	for _, v := range s.vals {
		if Equiv(v, key) {
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

func (s *Set) Cons(v interface{}) Conser {
	if s.Contains(v) {
		return s
	}
	return NewSet(append(s.vals, v)...)
}

func (s *Set) Disjoin(v interface{}) IPersistentSet {
	for i, val := range s.vals {
		if Equiv(val, v) {
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
		if Equiv(val, v) {
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

func (s *Set) Empty() IPersistentCollection {
	return emptySet.WithMeta(s.Meta()).(IPersistentCollection)
}

func (s *Set) String() string {
	return PrintString(s)
}

func (s *Set) Equals(v2 interface{}) bool {
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
	return NewSliceSeq(s.vals)
}

func (s *Set) Equiv(o any) bool {
	return apersistentsetEquiv(s, o)
}

func (s *Set) HashEq() uint32 {
	return apersistentsetHashEq(&s.hasheq, s)
}

func (s *Set) Meta() IPersistentMap {
	return s.meta
}

func (s *Set) WithMeta(meta IPersistentMap) interface{} {
	if meta == s.meta {
		return s
	}

	return &Set{
		meta: meta,
		vals: s.vals,
	}
}

func (s *Set) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientSet{Set: s}
}

type TransientSet struct {
	*Set
}

func (s *TransientSet) Conj(v interface{}) Conjer {
	return &TransientSet{Set: s.Set.Cons(v).(*Set)}
}

func (s *TransientSet) Persistent() IPersistentCollection {
	return s.Set
}
