package lang

// SortedMap wraps a Map and provides sorted iteration order by keys.
type SortedMap struct {
	m          IPersistentMap
	meta       IPersistentMap
	comparator IFn
}

type PersistentTreeMap = SortedMap

func CreatePersistentTreeMap(keyvals interface{}) interface{} {
	m := NewMap(seqToSlice(Seq(keyvals))...)
	return &SortedMap{m: m}
}

func CreatePersistentTreeMapWithComparator(comparator IFn, keyvals interface{}) interface{} {
	m := NewMap(seqToSlice(Seq(keyvals))...)
	return &SortedMap{m: m, comparator: comparator}
}

func (s *SortedMap) sortedKeys() []any {
	keys := make([]any, 0, s.m.Count())
	for seq := s.m.Seq(); seq != nil; seq = seq.Next() {
		e := seq.First().(IMapEntry)
		keys = append(keys, e.Key())
	}
	if s.comparator != nil {
		SortSlice(keys, s.comparator)
	} else {
		SortSlice(keys, FnFunc(func(args ...any) any {
			return LenientCompare(args[0], args[1])
		}))
	}
	return keys
}

// IPersistentMap methods

func (s *SortedMap) ValAt(key any) any {
	return s.m.ValAt(key)
}

func (s *SortedMap) ValAtDefault(key, def any) any {
	return s.m.ValAtDefault(key, def)
}

func (s *SortedMap) ContainsKey(key any) bool {
	return s.m.ContainsKey(key)
}

func (s *SortedMap) EntryAt(key any) IMapEntry {
	return s.m.EntryAt(key)
}

func (s *SortedMap) Assoc(key, val any) Associative {
	newM := s.m.Assoc(key, val).(IPersistentMap)
	return &SortedMap{m: newM, meta: s.meta, comparator: s.comparator}
}

func (s *SortedMap) AssocEx(key, val any) IPersistentMap {
	newM := s.m.AssocEx(key, val)
	return &SortedMap{m: newM, meta: s.meta, comparator: s.comparator}
}

func (s *SortedMap) Without(key any) IPersistentMap {
	newM := s.m.Without(key).(IPersistentMap)
	return &SortedMap{m: newM, meta: s.meta, comparator: s.comparator}
}

func (s *SortedMap) Cons(o any) Conser {
	newM := s.m.Cons(o).(IPersistentMap)
	return &SortedMap{m: newM, meta: s.meta, comparator: s.comparator}
}

func (s *SortedMap) Count() int {
	return s.m.Count()
}

func (s *SortedMap) xxx_counted() {}

func (s *SortedMap) IsEmpty() bool {
	return s.m.Count() == 0
}

func (s *SortedMap) Empty() IPersistentCollection {
	return &SortedMap{m: emptyMap, meta: s.meta, comparator: s.comparator}
}

func (s *SortedMap) Seq() ISeq {
	if s.m.Count() == 0 {
		return nil
	}
	keys := s.sortedKeys()
	entries := make([]any, len(keys))
	for i, k := range keys {
		entries[i] = &MapEntry{key: k, val: s.m.ValAt(k)}
	}
	return NewSliceSeq(entries)
}

func (s *SortedMap) Equiv(o any) bool {
	return apersistentmapEquiv(s, o)
}

func (s *SortedMap) Equals(o any) bool {
	return mapEquals(s, o)
}

func (s *SortedMap) Hash() uint32 {
	var h uint32
	return apersistentmapHash(&h, s)
}

func (s *SortedMap) HashEq() uint32 {
	var h uint32
	return apersistentmapHashEq(&h, s)
}

func (s *SortedMap) String() string {
	return PrintString(s)
}

func (s *SortedMap) Invoke(args ...any) any {
	return apersistentmapInvoke(s, args...)
}

func (s *SortedMap) ApplyTo(args ISeq) any {
	return afnApplyTo(s, args)
}

func (s *SortedMap) Meta() IPersistentMap {
	return s.meta
}

func (s *SortedMap) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

// Sorted interface
func (s *SortedMap) Comparator() IFn {
	return s.comparator
}

func (s *SortedMap) EntryKey(entry any) any {
	if e, ok := entry.(IMapEntry); ok {
		return e.Key()
	}
	return entry
}

// Reduce support
func (s *SortedMap) ReduceInit(f IFn, init any) any {
	ret := init
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		ret = f.Invoke(ret, seq.First())
		if IsReduced(ret) {
			return ret.(*Reduced).Deref()
		}
	}
	return ret
}

// RSeq satisfies the Reversible interface.
func (s *SortedMap) RSeq() ISeq {
	return s.Rseq()
}

// Rseq is an alias for RSeq, needed because FieldOrMethod capitalizes
// only the first letter of "rseq" to get "Rseq", not "RSeq".
func (s *SortedMap) Rseq() ISeq {
	if s.m.Count() == 0 {
		return nil
	}
	keys := s.sortedKeys()
	// reverse
	for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
		keys[i], keys[j] = keys[j], keys[i]
	}
	entries := make([]any, len(keys))
	for i, k := range keys {
		entries[i] = &MapEntry{key: k, val: s.m.ValAt(k)}
	}
	return NewSliceSeq(entries)
}
