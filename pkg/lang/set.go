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
	s := NewSet(seqToSlice(keys)...)
	return &SortedSet{Set: *s}
}

func CreatePersistentTreeSetWithComparator(comparator IFn, keys ISeq) any {
	s := NewSet(seqToSlice(keys)...)
	return &SortedSet{Set: *s, comparator: comparator}
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

func (s *Set) ReduceInit(f IFn, init any) any {
	ret := init
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		ret = f.Invoke(ret, seq.First())
		if IsReduced(ret) {
			return ret.(*Reduced).Deref()
		}
	}
	return ret
}

func (s *Set) Reduce(f IFn) any {
	seq := s.Seq()
	if seq == nil {
		return f.Invoke()
	}
	ret := seq.First()
	for seq = seq.Next(); seq != nil; seq = seq.Next() {
		ret = f.Invoke(ret, seq.First())
		if IsReduced(ret) {
			return ret.(*Reduced).Deref()
		}
	}
	return ret
}

func (s *Set) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientSet{Set: s}
}

type TransientSet struct {
	*Set
	persisted bool
}

func (s *TransientSet) ensureEditable() {
	if s.persisted {
		panic(NewIllegalStateError("transient used after persistent! call"))
	}
}

func (s *TransientSet) Conj(v any) Conjer {
	s.ensureEditable()
	s.Set = s.Set.Cons(v).(*Set)
	return s
}

func (s *TransientSet) Disjoin(v any) ITransientSet {
	s.ensureEditable()
	s.Set = s.Set.Disjoin(v).(*Set)
	return s
}

func (s *TransientSet) Persistent() IPersistentCollection {
	s.ensureEditable()
	s.persisted = true
	return s.Set
}

////////////////////////////////////////////////////////////////////////////////
// SortedSet

// SortedSet wraps a Set and provides sorted iteration order.
type SortedSet struct {
	Set
	comparator IFn // nil means default compare
}

type PersistentTreeSet = SortedSet

func (s *SortedSet) sortedElements() []any {
	elems := make([]any, 0, s.Count())
	for seq := s.Set.Seq(); seq != nil; seq = seq.Next() {
		elems = append(elems, seq.First())
	}
	if s.comparator != nil {
		SortSlice(elems, s.comparator)
	} else {
		SortSlice(elems, FnFunc(func(args ...any) any {
			return LenientCompare(args[0], args[1])
		}))
	}
	return elems
}

func (s *SortedSet) Seq() ISeq {
	if s.Count() == 0 {
		return nil
	}
	elems := s.sortedElements()
	return NewSliceSeq(elems)
}

func (s *SortedSet) Cons(v any) Conser {
	if s.Contains(v) {
		return s
	}
	inner := s.Set.Cons(v).(*Set)
	return &SortedSet{Set: *inner, comparator: s.comparator}
}

func (s *SortedSet) Disjoin(v any) IPersistentSet {
	if !s.Contains(v) {
		return s
	}
	inner := s.Set.Disjoin(v).(*Set)
	return &SortedSet{Set: *inner, comparator: s.comparator}
}

func (s *SortedSet) Empty() IPersistentCollection {
	return &SortedSet{Set: *emptySet, comparator: s.comparator}
}

func (s *SortedSet) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

// RSeq satisfies the Reversible interface.
func (s *SortedSet) RSeq() ISeq {
	return s.Rseq()
}

// Rseq is an alias for RSeq, needed because FieldOrMethod capitalizes
// only the first letter of "rseq" to get "Rseq", not "RSeq".
func (s *SortedSet) Rseq() ISeq {
	if s.Count() == 0 {
		return nil
	}
	elems := s.sortedElements()
	// reverse
	for i, j := 0, len(elems)-1; i < j; i, j = i+1, j-1 {
		elems[i], elems[j] = elems[j], elems[i]
	}
	return NewSliceSeq(elems)
}

func (s *SortedSet) Comparator() IFn {
	return s.comparator
}

func (s *SortedSet) EntryKey(entry any) any {
	return entry
}

func (s *SortedSet) ReduceInit(f IFn, init any) any {
	ret := init
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		ret = f.Invoke(ret, seq.First())
		if IsReduced(ret) {
			return ret.(*Reduced).Deref()
		}
	}
	return ret
}

func (s *SortedSet) Reduce(f IFn) any {
	seq := s.Seq()
	if seq == nil {
		return f.Invoke()
	}
	ret := seq.First()
	for seq = seq.Next(); seq != nil; seq = seq.Next() {
		ret = f.Invoke(ret, seq.First())
		if IsReduced(ret) {
			return ret.(*Reduced).Deref()
		}
	}
	return ret
}
