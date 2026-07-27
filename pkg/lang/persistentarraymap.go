package lang

import (
	"fmt"
)

const (
	arrayMapHashThreshold    = 16
	arrayMapKeywordThreshold = 128
	arrayMapInlineSize       = arrayMapHashThreshold - 2
	keywordMapDeltaMax       = 8

	// PersistentArrayMapInlineKeyValueCount lets the compiler preserve the
	// one-allocation constructor for maps whose entries fit inline.
	PersistentArrayMapInlineKeyValueCount = arrayMapInlineSize

	// PersistentArrayMapMaxKeywordKeyValueCount is the largest keyword-only
	// literal that should use the linear persistent-array-map representation.
	PersistentArrayMapMaxKeywordKeyValueCount = arrayMapKeywordThreshold
)

type (
	// Map represents a map of glojure values.
	Map struct {
		meta         IPersistentMap
		hash, hasheq uint32

		keyVals      []any
		keywordShape *KeywordMapShape
		keywordDelta *keywordMapDelta
	}

	// KeywordMapShape is an immutable key layout shared by maps emitted from
	// the same AOT keyword-map literal. Shaped maps store only their values.
	KeywordMapShape struct {
		keys []Keyword
	}

	// keywordMapDelta is a bounded immutable overlay on a shaped map's base
	// values. It gives compiler-emitted keyword maps structural sharing on
	// updates without imposing a trie on their fixed key layout.
	keywordMapDelta struct {
		prev  *keywordMapDelta
		value any
		index uint32
		depth uint8
	}

	// keywordMapUpdateStorage co-allocates a shaped-map version with its newest
	// immutable delta. The embedded Map points into its owning allocation, as
	// inlineMapStorage does for small array maps.
	keywordMapUpdateStorage struct {
		Map
		delta keywordMapDelta
	}

	// inlineMapStorage keeps small maps to one allocation without making every
	// Map carry unused inline capacity. A pointer to Map keeps its owner alive.
	inlineMapStorage struct {
		Map
		keyVals [arrayMapInlineSize]any
	}

	MapSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		keyVals []any
		shape   *KeywordMapShape
		shaped  *Map
		index   int
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

	_ ASeq        = (*MapKeySeq)(nil)
	_ IReduce     = (*MapKeySeq)(nil)
	_ IReduceInit = (*MapKeySeq)(nil)

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

	if !canBePersistentArrayMap(keyVals) {
		return NewPersistentHashMap(keyVals...)
	}

	return newArrayMap(keyVals)
}

// NewMapUniqueKeys constructs a map from compiler-owned key/value storage.
// Like Clojure's RT.mapUniqueKeys, it may retain the variadic backing array
// when the entries form a non-small persistent array map. Callers passing a
// slice with ... must not mutate that slice afterward.
func NewMapUniqueKeys(keyVals ...any) IPersistentMap {
	if len(keyVals) == 0 {
		return emptyMap
	}

	if len(keyVals)%2 != 0 {
		panic("invalid map. must have even number of inputs")
	}

	if !canBePersistentArrayMap(keyVals) {
		return NewPersistentHashMap(keyVals...)
	}

	if len(keyVals) <= arrayMapInlineSize {
		return newArrayMap(keyVals)
	}
	return &Map{keyVals: keyVals}
}

func NewKeywordMapShape(names ...string) *KeywordMapShape {
	keys := make([]Keyword, len(names))
	for i, name := range names {
		keys[i] = NewKeyword(name)
	}
	return &KeywordMapShape{keys: keys}
}

// NewStaticKeywordMap constructs a persistent map whose immutable keyword
// layout is shared with other maps from the same AOT literal.
func NewStaticKeywordMap(shape *KeywordMapShape, values ...any) IPersistentMap {
	if shape == nil || len(shape.keys) != len(values) {
		panic("invalid static keyword map shape")
	}
	if len(values) == 0 {
		return emptyMap
	}
	return &Map{keyVals: values, keywordShape: shape}
}

func (s *KeywordMapShape) indexOf(key Keyword) int {
	for i, candidate := range s.keys {
		if candidate == key {
			return i
		}
	}
	return -1
}

func newArrayMap(keyVals []any) *Map {
	if len(keyVals) <= arrayMapInlineSize {
		storage := &inlineMapStorage{}
		copy(storage.keyVals[:], keyVals)
		storage.Map.keyVals = storage.keyVals[:len(keyVals)]
		return &storage.Map
	}
	return &Map{keyVals: append([]any(nil), keyVals...)}
}

// canBePersistentArrayMap mirrors Clojure's PersistentArrayMap thresholds.
// Small maps stay array-backed regardless of key type. Larger maps may remain
// array-backed when every key beyond the general threshold is a keyword,
// whose identity comparison keeps linear lookup inexpensive.
func canBePersistentArrayMap(keyVals []any) bool {
	if len(keyVals) <= arrayMapHashThreshold {
		return true
	}
	if len(keyVals) > arrayMapKeywordThreshold {
		return false
	}
	for i := arrayMapHashThreshold; i < len(keyVals); i += 2 {
		if _, ok := keyVals[i].(Keyword); !ok {
			return false
		}
	}
	return true
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
		return m.valAtKeyword(kw, def)
	}
	if m.keywordShape != nil {
		return def
	}

	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], key) {
			return m.keyVals[i+1]
		}
	}

	return def
}

func (m *Map) valAtKeyword(key Keyword, def any) any {
	if m.keywordShape != nil {
		if i := m.keywordShape.indexOf(key); i >= 0 {
			return m.keywordValueAt(i)
		}
		return def
	}
	for i := 0; i < len(m.keyVals); i += 2 {
		if candidate, ok := m.keyVals[i].(Keyword); ok && key == candidate {
			return m.keyVals[i+1]
		}
	}
	return def
}

func (m *Map) keywordValueAt(index int) any {
	for delta := m.keywordDelta; delta != nil; delta = delta.prev {
		if int(delta.index) == index {
			return delta.value
		}
	}
	return m.keyVals[index]
}

func (m *Map) materializedKeywordValues() []any {
	values := append([]any(nil), m.keyVals...)
	var deltas [keywordMapDeltaMax]*keywordMapDelta
	count := 0
	for delta := m.keywordDelta; delta != nil; delta = delta.prev {
		deltas[count] = delta
		count++
	}
	for i := count - 1; i >= 0; i-- {
		delta := deltas[i]
		values[delta.index] = delta.value
	}
	return values
}

func (m *Map) EntryAt(k any) IMapEntry {
	if m.keywordShape != nil {
		kw, ok := k.(Keyword)
		if !ok {
			return nil
		}
		if i := m.keywordShape.indexOf(kw); i >= 0 {
			return NewMapEntry(kw, m.keywordValueAt(i))
		}
		return nil
	}
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], k) {
			return NewMapEntry(m.keyVals[i], m.keyVals[i+1])
		}
	}

	return nil
}

func (m *Map) clone() *Map {
	if m.keywordShape != nil {
		return &Map{
			meta:         m.meta,
			keyVals:      m.materializedKeywordValues(),
			keywordShape: m.keywordShape,
		}
	}
	cpy := newArrayMap(m.keyVals)
	cpy.meta = m.meta
	return cpy
}

func newKeywordMapUpdate(
	m *Map,
	prev *keywordMapDelta,
	value any,
	index int,
	depth uint8,
) *Map {
	storage := &keywordMapUpdateStorage{}
	storage.delta = keywordMapDelta{
		prev:  prev,
		value: value,
		index: uint32(index),
		depth: depth,
	}
	storage.Map = Map{
		meta:         m.meta,
		keyVals:      m.keyVals,
		keywordShape: m.keywordShape,
		keywordDelta: &storage.delta,
	}
	return &storage.Map
}

func (m *Map) Assoc(k, v any) Associative {
	if m.keywordShape != nil {
		if kw, ok := k.(Keyword); ok {
			if i := m.keywordShape.indexOf(kw); i >= 0 {
				if Identical(m.keywordValueAt(i), v) {
					return m
				}
				if m.keywordDelta != nil && int(m.keywordDelta.index) == i {
					return newKeywordMapUpdate(
						m,
						m.keywordDelta.prev,
						v,
						i,
						m.keywordDelta.depth,
					)
				}
				if m.keywordDelta == nil || m.keywordDelta.depth < keywordMapDeltaMax {
					depth := uint8(1)
					if m.keywordDelta != nil {
						depth = m.keywordDelta.depth + 1
					}
					return newKeywordMapUpdate(m, m.keywordDelta, v, i, depth)
				}
				newMap := m.clone()
				newMap.keyVals[i] = v
				return newMap
			}
		}
		keyVals := m.interleavedKeyVals(1)
		keyVals = append(keyVals, k, v)
		return NewMapUniqueKeys(keyVals...).(IObj).WithMeta(m.meta).(Associative)
	}
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equiv(m.keyVals[i], k) {
			if Identical(m.keyVals[i+1], v) {
				return m
			}
			newMap := m.clone()
			newMap.keyVals[i+1] = v
			return newMap
		}
	}
	threshold := arrayMapHashThreshold
	if _, ok := k.(Keyword); ok {
		threshold = arrayMapKeywordThreshold
	}
	if len(m.keyVals) < threshold {
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
	if m.keywordShape != nil {
		kw, ok := k.(Keyword)
		if !ok {
			return m
		}
		remove := m.keywordShape.indexOf(kw)
		if remove < 0 {
			return m
		}
		keyVals := make([]any, 0, (len(m.keyVals)-1)*2)
		for i, key := range m.keywordShape.keys {
			if i != remove {
				keyVals = append(keyVals, key, m.keywordValueAt(i))
			}
		}
		return NewMapUniqueKeys(keyVals...).(IObj).WithMeta(m.meta).(IPersistentMap)
	}
	newKeyVals := make([]any, 0, len(m.keyVals))
	for i := 0; i < len(m.keyVals); i += 2 {
		if !Equiv(m.keyVals[i], k) {
			newKeyVals = append(newKeyVals, m.keyVals[i], m.keyVals[i+1])
		}
	}
	return NewMap(newKeyVals...)
}

func (m *Map) Count() int {
	if m.keywordShape != nil {
		return len(m.keyVals)
	}
	return len(m.keyVals) / 2
}

func (m *Map) xxx_counted() {}

func (m *Map) Seq() ISeq {
	if len(m.keyVals) == 0 {
		return nil
	}
	if m.keywordShape != nil {
		return newKeywordMapSeq(m)
	}
	return NewMapSeq(m.keyVals)
}

func (m *Map) interleavedKeyVals(extraEntries int) []any {
	keyVals := make([]any, 0, (len(m.keyVals)+extraEntries)*2)
	for i, key := range m.keywordShape.keys {
		keyVals = append(keyVals, key, m.keywordValueAt(i))
	}
	return keyVals
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
	persisted bool
}

var (
	_ IPersistentMap = (*TransientMap)(nil)
	_ IMeta          = (*TransientMap)(nil)
	_ IFn            = (*TransientMap)(nil)
	_ IReduce        = (*TransientMap)(nil)
	_ IReduceInit    = (*TransientMap)(nil)
)

func (m *TransientMap) ensureEditable() {
	if m.persisted {
		panic(NewIllegalStateError("transient used after persistent! call"))
	}
}

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
	m.ensureEditable()
	m.IPersistentMap = m.IPersistentMap.Cons(v).(IPersistentMap)
	return m
}

func (m *TransientMap) Assoc(k, v any) Associative {
	m.ensureEditable()
	m.IPersistentMap = m.IPersistentMap.Assoc(k, v).(IPersistentMap)
	return m
}

func (m *TransientMap) Without(key any) IPersistentMap {
	m.ensureEditable()
	m.IPersistentMap = m.IPersistentMap.Without(key).(IPersistentMap)
	return m
}

func (m *TransientMap) Persistent() IPersistentCollection {
	m.ensureEditable()
	m.persisted = true
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

func newKeywordMapSeq(m *Map) *MapSeq {
	if m == nil || m.keywordShape == nil || len(m.keyVals) == 0 {
		return nil
	}
	return &MapSeq{shape: m.keywordShape, shaped: m}
}

// NewKeywordMapSeq constructs a sequence over an immutable shaped-map value
// slice. Map.Seq uses newKeywordMapSeq so delta overlays remain visible.
func NewKeywordMapSeq(shape *KeywordMapShape, values []any) *MapSeq {
	if shape == nil || len(values) == 0 {
		return nil
	}
	return newKeywordMapSeq(&Map{keyVals: values, keywordShape: shape})
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
	if s.shape != nil {
		return &MapEntry{
			key: s.shape.keys[s.index],
			val: s.shaped.keywordValueAt(s.index),
		}
	}
	return &MapEntry{
		key: s.keyVals[0],
		val: s.keyVals[1],
	}
}

func (s *MapSeq) Next() ISeq {
	if s.shape != nil {
		if s.index+1 >= len(s.shaped.keyVals) {
			return nil
		}
		return &MapSeq{
			shape:  s.shape,
			shaped: s.shaped,
			index:  s.index + 1,
		}
	}
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
	if s.shape != nil {
		return len(s.shaped.keyVals) - s.index
	}
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
	if s.shape != nil {
		acc := s.First()
		for i := s.index + 1; i < len(s.shaped.keyVals); i++ {
			acc = f.Invoke(acc, NewMapEntry(s.shape.keys[i], s.shaped.keywordValueAt(i)))
			if IsReduced(acc) {
				return acc.(IDeref).Deref()
			}
		}
		return acc
	}
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
	if s.shape != nil {
		acc := init
		for i := s.index; i < len(s.shaped.keyVals); i++ {
			acc = f.Invoke(acc, NewMapEntry(s.shape.keys[i], s.shaped.keywordValueAt(i)))
			if IsReduced(acc) {
				return acc.(IDeref).Deref()
			}
		}
		return acc
	}
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
	if s.shape != nil {
		return &MapSeq{
			shape:  s.shape,
			shaped: s.shaped,
			index:  s.index + n,
		}
	}
	return NewMapSeq(s.keyVals[n*2:])
}

////////////////////////////////////////////////////////////////////////////////

func NewMapKeySeq(s ISeq) ISeq {
	if IsNil(s) {
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

func (s *MapKeySeq) Reduce(f IFn) any {
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

func (s *MapKeySeq) ReduceInit(f IFn, init any) any {
	res := init
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

////////////////////////////////////////////////////////////////////////////////

func NewMapValSeq(s ISeq) ISeq {
	if IsNil(s) {
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
