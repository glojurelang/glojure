//go:generate go run ../cmd/gen-abstract-class/main.go -class APersistentMap -struct PersistentStructMap -receiver m
//go:generate go run ../cmd/gen-abstract-class/main.go -class ASeq -struct persistentStructMapSeq -receiver s
package value

import "fmt"

type (
	PersistentStructMap struct {
		meta IPersistentMap

		def  *PersistentStructMapDef
		vals []interface{}
		ext  IPersistentMap
	}

	PersistentStructMapDef struct {
		keys     ISeq
		keyslots IPersistentMap
	}

	persistentStructMapSeq struct {
		meta IPersistentMap
		i    int
		keys ISeq
		vals []interface{}
		ext  IPersistentMap
	}
)

var (
	_ IObj           = (*PersistentStructMap)(nil)
	_ IPersistentMap = (*PersistentStructMap)(nil)
	_ IFn            = (*PersistentStructMap)(nil)

	_ ISeq = (*persistentStructMapSeq)(nil)
	_ IObj = (*persistentStructMapSeq)(nil)
)

func ConstructPersistentStructMap(def *PersistentStructMapDef, valseq ISeq) *PersistentStructMap {
	vals := make([]interface{}, def.keyslots.Count())
	ext := emptyMap
	for i := 0; i < len(vals) && valseq != nil; valseq, i = valseq.Next(), i+1 {
		vals[i] = valseq.First()
	}
	if valseq != nil {
		panic(fmt.Errorf("too many arguments to struct constructor"))
	}
	return newPersistentStructMap(nil, def, vals, ext)
}

func CreatePersistentStructMapSlotMap(keys ISeq) *PersistentStructMapDef {
	if keys == nil {
		panic(fmt.Errorf("must supply keys"))
	}
	c := Count(keys)
	v := make([]interface{}, 2*c)
	i := 0
	for s := keys; s != nil; s, i = s.Next(), i+1 {
		v[2*i] = s.First()
		v[2*i+1] = i
	}
	return &PersistentStructMapDef{
		keys:     keys,
		keyslots: NewMap(v...),
	}
}

func newPersistentStructMap(meta IPersistentMap, def *PersistentStructMapDef, vals []interface{}, ext IPersistentMap) *PersistentStructMap {
	return &PersistentStructMap{
		meta: meta,
		def:  def,
		vals: vals,
		ext:  ext,
	}
}

func (m *PersistentStructMap) Meta() IPersistentMap {
	return m.meta
}

func (m *PersistentStructMap) WithMeta(meta IPersistentMap) interface{} {
	if Equal(m.meta, meta) {
		return m
	}
	cpy := *m
	cpy.meta = meta
	return &cpy
}

func (m *PersistentStructMap) Assoc(k interface{}, v interface{}) Associative {
	e := m.def.keyslots.EntryAt(k)
	if e == nil {
		return newPersistentStructMap(m.meta, m.def, m.vals, m.ext.Assoc(k, v).(IPersistentMap))
	}
	i := e.Val().(int)
	newVals := make([]interface{}, len(m.vals))
	copy(newVals, m.vals)
	newVals[i] = v
	return newPersistentStructMap(m.meta, m.def, newVals, m.ext)
}

func (m *PersistentStructMap) Count() int {
	return len(m.vals) + Count(m.ext)
}

func (m *PersistentStructMap) EntryAt(k interface{}) IMapEntry {
	e := m.def.keyslots.EntryAt(k)
	if e != nil {
		return NewMapEntry(e.Key(), m.vals[e.Val().(int)])
	}
	return m.ext.EntryAt(k)
}

func (m *PersistentStructMap) Seq() ISeq {
	return newPersistentStructMapSeq(nil, m.def.keys, m.vals, 0, m.ext)
}

func (m *PersistentStructMap) ValAtDefault(key, def interface{}) interface{} {
	if i, ok := m.def.keyslots.ValAt(key).(int); ok {
		return m.vals[i]
	}
	return m.ext.ValAtDefault(key, def)
}

func (m *PersistentStructMap) Without(k interface{}) IPersistentMap {
	e := m.def.keyslots.EntryAt(k)
	if e != nil {
		panic(fmt.Errorf("cannot remove struct key"))
	}
	newExt := m.ext.Without(k)
	if newExt == m.ext {
		return m
	}
	return newPersistentStructMap(m.meta, m.def, m.vals, newExt)
}

////////////////////////////////////////////////////////////////////////////////
// persistentStructMapSeq

func newPersistentStructMapSeq(meta IPersistentMap, keys ISeq, vals []interface{}, i int, ext IPersistentMap) *persistentStructMapSeq {
	return &persistentStructMapSeq{
		meta: meta,
		i:    i,
		keys: keys,
		vals: vals,
		ext:  ext,
	}
}

func (s *persistentStructMapSeq) First() interface{} {
	return NewMapEntry(s.keys.First(), s.vals[s.i])
}

func (s *persistentStructMapSeq) Next() ISeq {
	if len(s.vals) > s.i+1 {
		return newPersistentStructMapSeq(s.meta, s.keys.Next(), s.vals, s.i+1, s.ext)
	}
	return s.ext.Seq()
}
