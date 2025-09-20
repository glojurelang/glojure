package lang

import (
	"fmt"
)

const (
	hashmapThreshold = 16
)

type (
	// Map represents a map of glojure values.
	Map struct {
		meta         IPersistentMap
		hash, hasheq uint32

		keyVals []any
	}

	MapSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		keyVals []any
	}
	MapKeySeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		s ISeq
	}
	MapValSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		s ISeq
	}
)

var (
	_ APersistentMap = (*Map)(nil)
	_ IMeta          = (*Map)(nil)
	_ IObj           = (*Map)(nil)
	_ IFn            = (*Map)(nil)
	_ IReduce        = (*Map)(nil)
	_ IReduceInit    = (*Map)(nil)

	_ ASeq        = (*MapSeq)(nil)
	_ Counted     = (*MapSeq)(nil)
	_ IReduce     = (*MapSeq)(nil)
	_ IReduceInit = (*MapSeq)(nil)
	_ IDrop       = (*MapSeq)(nil)

	_ ASeq = (*MapKeySeq)(nil)

	_ ASeq        = (*MapValSeq)(nil)
	_ IReduce     = (*MapValSeq)(nil)
	_ IReduceInit = (*MapValSeq)(nil)

	emptyMap = &Map{}
)

////////////////////////////////////////////////////////////////////////////////
// Map

func NewMap(keyVals ...any) IPersistentMap {
	if len(keyVals) == 0 {
		return emptyMap
	}

	if len(keyVals)%2 != 0 {
		panic("invalid map. must have even number of inputs")
	}

	if len(keyVals) >= hashmapThreshold {
		return NewPersistentHashMap(keyVals...)
	}

	kv := make([]any, len(keyVals))
	copy(kv, keyVals)

	return &Map{
		keyVals: kv,
	}
}

func NewPersistentArrayMapAsIfByAssoc(init []any) IPersistentMap {
	complexPath := (len(init) & 1) == 1
	for i := 0; i < len(init) && !complexPath; i += 2 {
		for j := 0; j < i; j += 2 {
			if equalKey(init[i], init[j]) {
				complexPath = true
				break
			}
		}
	}

	if complexPath {
		return newPersistentArrayMapAsIfByAssocComplexPath(init)
	}

	return NewMap(init...)
}

func newPersistentArrayMapAsIfByAssocComplexPath(init []any) IPersistentMap {
	n := 0
	for i := 0; i < len(init); i += 2 {
		duplicateKey := false
		for j := 0; j < i; j += 2 {
			if equalKey(init[i], init[j]) {
				duplicateKey = true
				break
			}
		}
		if !duplicateKey {
			n += 2
		}
	}

	if n < len(init) {
		nodups := make([]any, n)
		m := 0
		for i := 0; i < len(init); i += 2 {
			duplicateKey := false
			for j := 0; j < m; j += 2 {
				if equalKey(init[i], nodups[j]) {
					duplicateKey = true
					break
				}
			}
			if duplicateKey {
				continue
			}

			var j int
			for j = len(init) - 2; j >= i; j -= 2 {
				if equalKey(init[i], init[j]) {
					break
				}
			}
			nodups[m] = init[i]
			nodups[m+1] = init[j+1]
			m += 2
		}
		if m != n {
			panic(fmt.Errorf("internal error: m=%d", m))
		}
		init = nodups
	}
	return NewMap(init...)
}

func (m *Map) ValAt(key any) any {
	return m.ValAtDefault(key, nil)
}

func (m *Map) ValAtDefault(key, def any) any {
	if kw, ok := key.(Keyword); ok {
		for i := 0; i < len(m.keyVals); i += 2 {
			if kw == m.keyVals[i] {
				return m.keyVals[i+1]
			}
		}
		return def
	}

	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], key) {
			return m.keyVals[i+1]
		}
	}

	return def
}

func (m *Map) EntryAt(k any) IMapEntry {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], k) {
			return NewMapEntry(m.keyVals[i], m.keyVals[i+1])
		}
	}

	return nil
}

func (m *Map) clone() *Map {
	cpy := *m
	cpy.keyVals = make([]any, len(m.keyVals))
	copy(cpy.keyVals, m.keyVals)
	return &cpy
}

func (m *Map) Assoc(k, v any) Associative {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], k) {
			newMap := m.clone()
			newMap.keyVals[i+1] = v
			return newMap
		}
	}
	if len(m.keyVals) < hashmapThreshold {
		newMap := m.clone()
		newMap.keyVals = append(newMap.keyVals, k, v)
		return newMap
	}
	newMap := NewPersistentHashMap(m.keyVals...).(*PersistentHashMap).WithMeta(m.meta).(Associative)
	return newMap.Assoc(k, v)
}

func (m *Map) AssocEx(k, v any) IPersistentMap {
	return apersistentmapAssocEx(m, k, v)
}

func (m *Map) Without(k any) IPersistentMap {
	newKeyVals := make([]any, 0, len(m.keyVals))
	for i := 0; i < len(m.keyVals); i += 2 {
		if !Equiv(m.keyVals[i], k) {
			newKeyVals = append(newKeyVals, m.keyVals[i], m.keyVals[i+1])
		}
	}
	return NewMap(newKeyVals...)
}

func (m *Map) Count() int {
	return len(m.keyVals) / 2
}

func (m *Map) xxx_counted() {}

func (m *Map) Seq() ISeq {
	if len(m.keyVals) == 0 {
		return nil
	}
	return NewMapSeq(m.keyVals)
}

func (m *Map) Empty() IPersistentCollection {
	return emptyMap.WithMeta(m.meta).(IPersistentCollection)
}

func (m *Map) String() string {
	return apersistentmapString(m)
}

func (m *Map) Meta() IPersistentMap {
	return m.meta
}

func (m *Map) WithMeta(meta IPersistentMap) any {
	if m.meta == meta {
		return m
	}
	cpy := *m
	cpy.meta = meta
	return &cpy
}

func (m *Map) ApplyTo(args ISeq) any {
	return afnApplyTo(m, args)
}

func (m *Map) Invoke(args ...any) any {
	return apersistentmapInvoke(m, args...)
}

func (m *Map) Cons(x any) Conser {
	return apersistentmapCons(m, x)
}

func (m *Map) ContainsKey(k any) bool {
	return apersistentmapContainsKey(m, k)
}

func (m *Map) Equiv(o any) bool {
	return apersistentmapEquiv(m, o)
}

func (m *Map) Hash() uint32 {
	return apersistentmapHash(&m.hash, m)
}

func (m *Map) HashEq() uint32 {
	return apersistentmapHashEq(&m.hasheq, m)
}

func (m *Map) Reduce(f IFn) any {
	if m.Count() == 0 {
		return f.Invoke()
	}
	var res any
	first := true
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		if first {
			res = seq.First()
			first = false
			continue
		}
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (m *Map) ReduceInit(f IFn, init any) any {
	res := init
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (m *Map) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientMap{IPersistentMap: m}
}

////////////////////////////////////////////////////////////////////////////////
// Transient

type TransientMap struct {
	IPersistentMap
}

var (
	_ IPersistentMap = (*TransientMap)(nil)
	_ IMeta          = (*TransientMap)(nil)
	_ IFn            = (*TransientMap)(nil)
	_ IReduce        = (*TransientMap)(nil)
	_ IReduceInit    = (*TransientMap)(nil)
)

func (m *TransientMap) Meta() IPersistentMap {
	return m.IPersistentMap.(IMeta).Meta()
}

func (m *TransientMap) ApplyTo(args ISeq) any {
	return m.IPersistentMap.(IFn).ApplyTo(args)
}

func (m *TransientMap) Invoke(args ...any) any {
	return m.IPersistentMap.(IFn).Invoke(args...)
}

func (m *TransientMap) Reduce(f IFn) any {
	return m.IPersistentMap.(IReduce).Reduce(f)
}

func (m *TransientMap) ReduceInit(f IFn, init any) any {
	return m.IPersistentMap.(IReduceInit).ReduceInit(f, init)
}

func (m *TransientMap) Conj(v any) Conjer {
	return &TransientMap{IPersistentMap: m.IPersistentMap.Cons(v).(IPersistentMap)}
}

func (m *TransientMap) Assoc(k, v any) Associative {
	return &TransientMap{IPersistentMap: m.IPersistentMap.Assoc(k, v).(IPersistentMap)}
}

func (m *TransientMap) Persistent() IPersistentCollection {
	return m.IPersistentMap
}

////////////////////////////////////////////////////////////////////////////////
// Map ISeqs

func NewMapSeq(kvs []any) *MapSeq {
	if len(kvs) == 0 {
		return nil
	}
	return &MapSeq{
		keyVals: kvs,
	}
}

func (s *MapSeq) xxx_sequential() {}

func (s *MapSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *MapSeq) WithMeta(meta IPersistentMap) any {
	if s.meta == meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *MapSeq) String() string {
	return aseqString(s)
}

func (s *MapSeq) Seq() ISeq {
	return s
}

func (s *MapSeq) First() any {
	return &MapEntry{
		key: s.keyVals[0],
		val: s.keyVals[1],
	}
}

func (s *MapSeq) Next() ISeq {
	if len(s.keyVals) <= 2 {
		return nil
	}
	return &MapSeq{
		keyVals: s.keyVals[2:],
	}
}

func (s *MapSeq) More() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (s *MapSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *MapSeq) Count() int {
	return len(s.keyVals) / 2
}

func (s *MapSeq) xxx_counted() {}

func (s *MapSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *MapSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *MapSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *MapSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *MapSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *MapSeq) Reduce(f IFn) any {
	if len(s.keyVals) == 0 {
		return f.Invoke()
	}
	acc := s.First()
	for i := 2; i < len(s.keyVals); i += 2 {
		acc = f.Invoke(acc, NewMapEntry(s.keyVals[i], s.keyVals[i+1]))
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}

func (s *MapSeq) ReduceInit(f IFn, init any) any {
	acc := init
	for i := 0; i < len(s.keyVals); i += 2 {
		acc = f.Invoke(acc, NewMapEntry(s.keyVals[i], s.keyVals[i+1]))
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}

func (s *MapSeq) Drop(n int) Sequential {
	if n >= s.Count() {
		return nil
	}
	return NewMapSeq(s.keyVals[n*2:])
}

////////////////////////////////////////////////////////////////////////////////

func NewMapKeySeq(s ISeq) ISeq {
	if s == nil {
		return nil
	}
	return &MapKeySeq{s: s}
}

func (s *MapKeySeq) Meta() IPersistentMap {
	return s.meta
}

func (s *MapKeySeq) WithMeta(meta IPersistentMap) any {
	if s.meta == meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *MapKeySeq) String() string {
	return aseqString(s)
}

func (s *MapKeySeq) xxx_sequential() {}

func (s *MapKeySeq) Seq() ISeq {
	return s
}

func (s *MapKeySeq) First() any {
	return s.s.First().(*MapEntry).Key()
}

func (s *MapKeySeq) Next() ISeq {
	return NewMapKeySeq(s.s.Next())
}

func (s *MapKeySeq) More() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (s *MapKeySeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *MapKeySeq) Count() int {
	return aseqCount(s)
}

func (s *MapKeySeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *MapKeySeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *MapKeySeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *MapKeySeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *MapKeySeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

////////////////////////////////////////////////////////////////////////////////

func NewMapValSeq(s ISeq) ISeq {
	if s == nil {
		return nil
	}
	return &MapValSeq{s: s}
}

func (s *MapValSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *MapValSeq) WithMeta(meta IPersistentMap) any {
	if s.meta == meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *MapValSeq) String() string {
	return aseqString(s)
}

func (s *MapValSeq) xxx_sequential() {}

func (s *MapValSeq) Seq() ISeq {
	return s
}

func (s *MapValSeq) First() any {
	return s.s.First().(*MapEntry).Val()
}

func (s *MapValSeq) Next() ISeq {
	return NewMapValSeq(s.s.Next())
}

func (s *MapValSeq) More() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (s *MapValSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *MapValSeq) Count() int {
	return aseqCount(s)
}

func (s *MapValSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *MapValSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *MapValSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *MapValSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *MapValSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *MapValSeq) Reduce(f IFn) any {
	count := 0
	var res any
	first := true
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		count++
		if first {
			res = seq.First()
			first = false
			continue
		}
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	if count == 0 {
		return f.Invoke()
	}
	return res
}

func (s *MapValSeq) ReduceInit(f IFn, init any) any {
	res := init
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}
