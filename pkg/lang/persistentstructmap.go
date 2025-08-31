package lang

import "fmt"

type (
	PersistentStructMap struct {
		meta         IPersistentMap
		hash, hasheq uint32

		def  *PersistentStructMapDef
		vals []any
		ext  IPersistentMap
	}

	PersistentStructMapDef struct {
		keys     ISeq
		keyslots IPersistentMap
	}

	persistentStructMapSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		i    int
		keys ISeq
		vals []any
		ext  IPersistentMap
	}
)

var (
	_ APersistentMap = (*PersistentStructMap)(nil)
	_ IObj           = (*PersistentStructMap)(nil)
	_ IPersistentMap = (*PersistentStructMap)(nil)
	_ IFn            = (*PersistentStructMap)(nil)

	_ ASeq = (*persistentStructMapSeq)(nil)
	_ ISeq = (*persistentStructMapSeq)(nil)
	_ IObj = (*persistentStructMapSeq)(nil)

	emptyPersistentStructMap = newPersistentStructMap(nil, nil, nil, emptyMap)
)

func ConstructPersistentStructMap(def *PersistentStructMapDef, valseq ISeq) *PersistentStructMap {
	vals := make([]any, def.keyslots.Count())
	ext := emptyMap
	for i := 0; i < len(vals) && valseq != nil; valseq, i = valseq.Next(), i+1 {
		vals[i] = valseq.First()
	}
	if valseq != nil {
		panic(fmt.Errorf("too many arguments to struct constructor"))
	}
	return newPersistentStructMap(nil, def, vals, ext)
}

/*
Object[] vals = new Object[def.keyslots.count()];
IPersistentMap ext = PersistentHashMap.EMPTY;
for(; keyvals != null; keyvals = keyvals.next().next())

	{
	if(keyvals.next() == null)
		throw new IllegalArgumentException(String.format("No value supplied for key: %s", keyvals.first()));
	Object k = keyvals.first();
	Object v = RT.second(keyvals);
	Map.Entry e = def.keyslots.entryAt(k);
	if(e != null)
		vals[(Integer) e.getValue()] = v;
	else
		ext = ext.assoc(k, v);
	}

return new PersistentStructMap(null, def, vals, ext);
*/
func CreatePersistentStructMap(def *PersistentStructMapDef, keyvals ISeq) *PersistentStructMap {
	vals := make([]any, def.keyslots.Count())
	var ext IPersistentMap = emptyMap
	for ; keyvals != nil; keyvals = keyvals.Next().Next() {
		if keyvals.Next() == nil {
			panic(fmt.Errorf("no value supplied for key: %v", keyvals.First()))
		}
		k := keyvals.First()
		v := First(Rest(keyvals))
		e := def.keyslots.EntryAt(k)
		if e != nil {
			vals[e.Val().(int)] = v
		} else {
			ext = ext.Assoc(k, v).(IPersistentMap)
		}
	}
	return newPersistentStructMap(nil, def, vals, ext)
}

func CreatePersistentStructMapSlotMap(keys ISeq) *PersistentStructMapDef {
	if keys == nil {
		panic(fmt.Errorf("must supply keys"))
	}
	c := Count(keys)
	v := make([]any, 2*c)
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

func newPersistentStructMap(meta IPersistentMap, def *PersistentStructMapDef, vals []any, ext IPersistentMap) *PersistentStructMap {
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

func (m *PersistentStructMap) WithMeta(meta IPersistentMap) any {
	if m.meta == meta {
		return m
	}
	cpy := *m
	cpy.meta = meta
	return &cpy
}

func (m *PersistentStructMap) Assoc(k any, v any) Associative {
	e := m.def.keyslots.EntryAt(k)
	if e == nil {
		return newPersistentStructMap(m.meta, m.def, m.vals, m.ext.Assoc(k, v).(IPersistentMap))
	}
	i := e.Val().(int)
	newVals := make([]any, len(m.vals))
	copy(newVals, m.vals)
	newVals[i] = v
	return newPersistentStructMap(m.meta, m.def, newVals, m.ext)
}

func (m *PersistentStructMap) Count() int {
	return len(m.vals) + Count(m.ext)
}

func (m *PersistentStructMap) EntryAt(k any) IMapEntry {
	e := m.def.keyslots.EntryAt(k)
	if e != nil {
		return NewMapEntry(e.Key(), m.vals[e.Val().(int)])
	}
	return m.ext.EntryAt(k)
}

func (m *PersistentStructMap) Seq() ISeq {
	if m.Count() == 0 {
		return nil
	}
	return newPersistentStructMapSeq(nil, m.def.keys, m.vals, 0, m.ext)
}

func (m *PersistentStructMap) Empty() IPersistentCollection {
	return emptyPersistentStructMap.WithMeta(m.meta).(IPersistentCollection)
}

func (m *PersistentStructMap) ValAtDefault(key, def any) any {
	if i, ok := m.def.keyslots.ValAt(key).(int); ok {
		return m.vals[i]
	}
	return m.ext.ValAtDefault(key, def)
}

func (m *PersistentStructMap) Without(k any) IPersistentMap {
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

func (m *PersistentStructMap) ApplyTo(args ISeq) any {
	return afnApplyTo(m, args)
}

func (m *PersistentStructMap) Invoke(args ...any) any {
	return apersistentmapInvoke(m, args...)
}

func (m *PersistentStructMap) AssocEx(k, v any) IPersistentMap {
	return apersistentmapAssocEx(m, k, v)
}

func (m *PersistentStructMap) Cons(x any) Conser {
	return apersistentmapCons(m, x)
}

func (m *PersistentStructMap) ContainsKey(k any) bool {
	return apersistentmapContainsKey(m, k)
}

func (m *PersistentStructMap) Equiv(o any) bool {
	return apersistentmapEquiv(m, o)
}

func (m *PersistentStructMap) HashEq() uint32 {
	return apersistentmapHashEq(&m.hasheq, m)
}

func (m *PersistentStructMap) ValAt(key any) any {
	return m.ValAtDefault(key, nil)
}

////////////////////////////////////////////////////////////////////////////////
// persistentStructMapSeq

func newPersistentStructMapSeq(meta IPersistentMap, keys ISeq, vals []any, i int, ext IPersistentMap) *persistentStructMapSeq {
	return &persistentStructMapSeq{
		meta: meta,
		i:    i,
		keys: keys,
		vals: vals,
		ext:  ext,
	}
}

func (s *persistentStructMapSeq) First() any {
	return NewMapEntry(s.keys.First(), s.vals[s.i])
}

func (s *persistentStructMapSeq) Next() ISeq {
	if len(s.vals) > s.i+1 {
		return newPersistentStructMapSeq(s.meta, s.keys.Next(), s.vals, s.i+1, s.ext)
	}
	return s.ext.Seq()
}

func (s *persistentStructMapSeq) More() ISeq {
	return aseqMore(s)
}

func (s *persistentStructMapSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *persistentStructMapSeq) Count() int {
	return aseqCount(s)
}

func (s *persistentStructMapSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *persistentStructMapSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *persistentStructMapSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *persistentStructMapSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *persistentStructMapSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *persistentStructMapSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *persistentStructMapSeq) WithMeta(meta IPersistentMap) any {
	if s.meta == meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *persistentStructMapSeq) Seq() ISeq {
	return s
}

func (s *persistentStructMapSeq) String() string {
	return aseqString(s)
}

func (s *persistentStructMapSeq) xxx_sequential() {}
