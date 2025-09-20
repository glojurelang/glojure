package lang

import (
	"fmt"
)

// Set represents a map of glojure values.
type Set struct {
	meta         IPersistentMap
	hash, hasheq uint32

	hashMap IPersistentMap
}

type PersistentHashSet = Set

func CreatePersistentTreeSet(keys ISeq) any {
	// TODO: implement
	return NewSet(seqToSlice(keys)...)
}

func CreatePersistentTreeSetWithComparator(comparator IFn, keys ISeq) any {
	// TODO: implement
	return NewSet(seqToSlice(keys)...)
}

func NewSet(vals ...any) *Set {
	set, err := NewSet2(vals...)
	if err != nil {
		panic(err)
	}
	return set
}

func NewSet2(vals ...any) (*Set, error) {
	set := &Set{
		hashMap: NewPersistentHashMap(),
	}
	for i := 0; i < len(vals); i++ {
		val := vals[i]
		set.hashMap = set.hashMap.Assoc(val, true).(IPersistentMap)
	}

	return set, nil
}

var (
	_ APersistentSet        = (*Set)(nil)
	_ IObj                  = (*Set)(nil)
	_ IPersistentCollection = (*Set)(nil)

	emptySet = NewSet()
)

func (s *Set) Get(key any) any {
	val := s.hashMap.ValAt(key)
	if val == true {
		return key
	}
	return nil
}

func (s *Set) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(fmt.Errorf("set apply expects 1 argument, got %d", len(args)))
	}

	return s.Get(args[0])
}

func (s *Set) ApplyTo(args ISeq) any {
	return s.Invoke(seqToSlice(args)...)
}

func (s *Set) Cons(v any) Conser {
	if s.Contains(v) {
		return s
	}
	return &Set{
		meta:    s.meta,
		hashMap: s.hashMap.Assoc(v, true).(IPersistentMap),
	}
}

func (s *Set) Disjoin(v any) IPersistentSet {
	if !s.Contains(v) {
		return s
	}
	return &Set{
		meta:    s.meta,
		hashMap: s.hashMap.Without(v).(IPersistentMap),
	}
}

func (s *Set) Contains(v any) bool {
	return s.hashMap.ContainsKey(v)
}

func (s *Set) Count() int {
	return s.hashMap.Count()
}

func (s *Set) xxx_counted() {}

func (s *Set) IsEmpty() bool {
	return s.Count() == 0
}

func (s *Set) Empty() IPersistentCollection {
	return emptySet.WithMeta(s.Meta()).(IPersistentCollection)
}

func (s *Set) String() string {
	return PrintString(s)
}

func (s *Set) Equals(v2 any) bool {
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
	if s.hashMap.Count() == 0 {
		return nil
	}
	return NewMapKeySeq(Seq(s.hashMap))
}

func (s *Set) Equiv(o any) bool {
	return apersistentsetEquiv(s, o)
}

func (s *Set) Hash() uint32 {
	return apersistentsetHash(&s.hash, s)
}

func (s *Set) HashEq() uint32 {
	return apersistentsetHashEq(&s.hasheq, s)
}

func (s *Set) Meta() IPersistentMap {
	return s.meta
}

func (s *Set) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}

	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *Set) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientSet{Set: s}
}

type TransientSet struct {
	*Set
}

func (s *TransientSet) Conj(v any) Conjer {
	return &TransientSet{Set: s.Set.Cons(v).(*Set)}
}

func (s *TransientSet) Persistent() IPersistentCollection {
	return s.Set
}
