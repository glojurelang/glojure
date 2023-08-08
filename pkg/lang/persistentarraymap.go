//go:generate go run ../../cmd/gen-abstract-class/main.go -class APersistentMap -struct Map -receiver m
package lang

import "fmt"

const (
	hashmapThreshold = 16
)

type (
	// Map represents a map of glojure values.
	Map struct {
		meta    IPersistentMap
		keyVals []interface{}
	}

	MapSeq struct {
		m          *Map
		entryIndex int
	}
	MapKeySeq struct {
		s ISeq
	}
	MapValSeq struct {
		s ISeq
	}
)

var (
	_ IPersistentMap = (*Map)(nil)
	_ IMeta          = (*Map)(nil)
	_ IObj           = (*Map)(nil)
	_ IFn            = (*Map)(nil)
	_ IReduce        = (*Map)(nil)
	_ IReduceInit    = (*Map)(nil)

	_ IReduce     = (*MapValSeq)(nil)
	_ IReduceInit = (*MapValSeq)(nil)

	emptyMap = &Map{}
)

////////////////////////////////////////////////////////////////////////////////
// Map

func NewMap(keyVals ...interface{}) IPersistentMap {
	if len(keyVals) == 0 {
		return emptyMap
	}

	if len(keyVals)%2 != 0 {
		panic("invalid map. must have even number of inputs")
	}

	if len(keyVals) >= hashmapThreshold {
		return NewPersistentHashMap(keyVals...)
	}

	kv := make([]interface{}, len(keyVals))
	copy(kv, keyVals)

	return &Map{
		keyVals: kv,
	}
}

func NewPersistentArrayMapAsIfByAssoc(init []interface{}) IPersistentMap {
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

func newPersistentArrayMapAsIfByAssocComplexPath(init []interface{}) IPersistentMap {
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
		nodups := make([]interface{}, n)
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

func (m *Map) ValAtDefault(key, def interface{}) interface{} {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equal(m.keyVals[i], key) {
			return m.keyVals[i+1]
		}
	}

	return def
}

func (m *Map) EntryAt(k interface{}) IMapEntry {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equal(m.keyVals[i], k) {
			return NewMapEntry(m.keyVals[i], m.keyVals[i+1])
		}
	}

	return nil
}

func (m *Map) clone() *Map {
	cpy := *m
	cpy.keyVals = make([]interface{}, len(m.keyVals))
	copy(cpy.keyVals, m.keyVals)
	return &cpy
}

func (m *Map) Assoc(k, v interface{}) Associative {
	for i := 0; i < len(m.keyVals); i += 2 {
		if Equal(m.keyVals[i], k) {
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

func (m *Map) Without(k interface{}) IPersistentMap {
	newKeyVals := make([]interface{}, 0, len(m.keyVals))
	for i := 0; i < len(m.keyVals); i += 2 {
		if !Equal(m.keyVals[i], k) {
			newKeyVals = append(newKeyVals, m.keyVals[i], m.keyVals[i+1])
		}
	}
	return NewMap(newKeyVals...)
}

func (m *Map) Count() int {
	return len(m.keyVals) / 2
}

func (m *Map) Seq() ISeq {
	return NewMapSeq(m)
}

func (m *Map) Empty() IPersistentCollection {
	return emptyMap.WithMeta(m.meta).(IPersistentCollection)
}

func (m *Map) String() string {
	return PrintString(m)
}

func (m *Map) Meta() IPersistentMap {
	return m.meta
}

func (m *Map) WithMeta(meta IPersistentMap) interface{} {
	if m.meta == meta {
		return m
	}
	cpy := *m
	cpy.meta = meta
	return &cpy
}

func (m *Map) Reduce(f IFn) interface{} {
	if m.Count() == 0 {
		return f.Invoke()
	}
	var res interface{}
	first := true
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		if first {
			res = seq.First()
			first = false
			continue
		}
		res = f.Invoke(res, seq.First())
	}
	return res
}

func (m *Map) ReduceInit(f IFn, init interface{}) interface{} {
	res := init
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
	}
	return res
}

func (m *Map) AsTransient() ITransientCollection {
	// TODO: implement transients
	return &TransientMap{Map: m}
}

////////////////////////////////////////////////////////////////////////////////
// Transient

type TransientMap struct {
	*Map
}

var (
	_ IPersistentMap = (*TransientMap)(nil)
	_ IMeta          = (*TransientMap)(nil)
	_ IFn            = (*TransientMap)(nil)
	_ IReduce        = (*TransientMap)(nil)
	_ IReduceInit    = (*TransientMap)(nil)
)

func (m *TransientMap) Conj(v interface{}) Conjer {
	return &TransientMap{Map: m.Map.Conj(v).(*Map)}
}

func (m *TransientMap) Assoc(k, v interface{}) Associative {
	return &TransientMap{Map: m.Map.Assoc(k, v).(*Map)}
}

func (m *TransientMap) Persistent() IPersistentCollection {
	return m.Map
}

////////////////////////////////////////////////////////////////////////////////
// Map ISeqs

func NewMapSeq(m *Map) ISeq {
	if m == nil || m.Count() == 0 {
		return nil
	}
	return &MapSeq{
		m:          m,
		entryIndex: 0,
	}
}

func (s *MapSeq) xxx_sequential() {}

func (s *MapSeq) Seq() ISeq {
	return s
}

func (s *MapSeq) First() interface{} {
	return &MapEntry{
		key: s.m.keyVals[2*s.entryIndex],
		val: s.m.keyVals[2*s.entryIndex+1],
	}
}

func (s *MapSeq) Next() ISeq {
	if s.entryIndex+1 >= s.m.Count() {
		return nil
	}
	return &MapSeq{
		m:          s.m,
		entryIndex: s.entryIndex + 1,
	}
}

func (s *MapSeq) More() ISeq {
	nxt := s.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func NewMapKeySeq(s ISeq) ISeq {
	if s == nil {
		return nil
	}
	return &MapKeySeq{s}
}

func (s *MapKeySeq) xxx_sequential() {}

func (s *MapKeySeq) Seq() ISeq {
	return s
}

func (s *MapKeySeq) First() interface{} {
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

func NewMapValSeq(s ISeq) ISeq {
	if s == nil {
		return nil
	}
	return &MapValSeq{s}
}

func (s *MapValSeq) xxx_sequential() {}

func (s *MapValSeq) Seq() ISeq {
	return s
}

func (s *MapValSeq) First() interface{} {
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

func (s *MapValSeq) Reduce(f IFn) interface{} {
	count := 0
	var res interface{}
	first := true
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		count++
		if first {
			res = seq.First()
			first = false
			continue
		}
		res = f.Invoke(res, seq.First())
	}
	if count == 0 {
		return f.Invoke()
	}
	return res
}

func (s *MapValSeq) ReduceInit(f IFn, init interface{}) interface{} {
	res := init
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
	}
	return res
}
