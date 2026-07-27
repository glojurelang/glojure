package lang

import "fmt"

// transientMapEdit is the ownership token shared by a transient map and every
// HAMT node it has made editable. Pointer identity determines whether a node
// may be changed in place.
type transientMapEdit struct {
	persisted bool
}

// TransientMap keeps a stable public identity while its representation changes
// from a small mutable array map to an edit-token-owned HAMT.
type TransientMap struct {
	edit *transientMapEdit

	// array is non-nil while the map uses the small-map representation.
	array []any

	count    int
	root     Node
	leafFlag Box
}

var (
	_ ITransientMap = (*TransientMap)(nil)
	_ IFn           = (*TransientMap)(nil)
)

func newTransientArrayMap(m *Map) *TransientMap {
	var keyVals []any
	if m.keywordShape != nil {
		keyVals = m.interleavedKeyVals(0)
	} else {
		keyVals = append([]any(nil), m.keyVals...)
	}
	capacity := arrayMapHashThreshold
	if len(keyVals) > capacity {
		capacity = len(keyVals)
	}
	array := make([]any, len(keyVals), capacity)
	copy(array, keyVals)
	return &TransientMap{
		edit:  &transientMapEdit{},
		array: array,
		count: len(keyVals) / 2,
	}
}

func newTransientHashMap(m *PersistentHashMap) *TransientMap {
	return &TransientMap{
		edit:  &transientMapEdit{},
		count: m.count,
		root:  m.root,
	}
}

func (m *TransientMap) ensureEditable() {
	if m.edit == nil || m.edit.persisted {
		panic(NewIllegalStateError("transient used after persistent! call"))
	}
}

func (m *TransientMap) Assoc(key, value any) ITransientAssociative {
	m.ensureEditable()
	if m.array != nil {
		return m.assocArray(key, value)
	}
	m.assocHAMT(key, value)
	return m
}

func (m *TransientMap) assocArray(key, value any) *TransientMap {
	if index := m.arrayIndex(key); index >= 0 {
		if !Identical(m.array[index+1], value) {
			m.array[index+1] = value
		}
		return m
	}

	threshold := arrayMapHashThreshold
	if _, ok := key.(Keyword); ok {
		threshold = arrayMapKeywordThreshold
	}
	if len(m.array) >= threshold {
		m.promoteArray()
		m.assocHAMT(key, value)
		return m
	}

	if len(m.array)+2 > cap(m.array) {
		capacity := threshold
		if capacity < len(m.array)+2 {
			capacity = len(m.array) + 2
		}
		grown := make([]any, len(m.array), capacity)
		copy(grown, m.array)
		m.array = grown
	}
	m.array = append(m.array, key, value)
	m.count++
	return m
}

func (m *TransientMap) assocHAMT(key, value any) {
	m.leafFlag.val = nil
	root := m.root
	if root == nil {
		root = emptyIndexedNode
	}
	m.root = root.assocTransient(
		m.edit,
		0,
		HashEq(key),
		key,
		value,
		&m.leafFlag,
	)
	if m.leafFlag.val != nil {
		m.count++
	}
}

func (m *TransientMap) Without(key any) ITransientMap {
	m.ensureEditable()
	if m.array != nil {
		index := m.arrayIndex(key)
		if index < 0 {
			return m
		}
		last := len(m.array) - 2
		m.array[index] = m.array[last]
		m.array[index+1] = m.array[last+1]
		m.array[last] = nil
		m.array[last+1] = nil
		m.array = m.array[:last]
		m.count--
		return m
	}
	if m.root == nil {
		return m
	}
	m.leafFlag.val = nil
	m.root = m.root.withoutTransient(
		m.edit,
		0,
		HashEq(key),
		key,
		&m.leafFlag,
	)
	if m.leafFlag.val != nil {
		m.count--
	}
	return m
}

func (m *TransientMap) Conj(value any) Conjer {
	m.ensureEditable()
	switch value := value.(type) {
	case IMapEntry:
		m.Assoc(value.Key(), value.Val())
		return m
	case IPersistentVector:
		if value.Count() != 2 {
			panic(NewIllegalArgumentError("vector arg to map conj must be a pair"))
		}
		m.Assoc(MustNth(value, 0), MustNth(value, 1))
		return m
	}
	for seq := Seq(value); seq != nil; seq = seq.Next() {
		entry, ok := seq.First().(IMapEntry)
		if !ok {
			panic(NewIllegalArgumentError(
				fmt.Sprintf("map conj expects map entries, got %T", seq.First()),
			))
		}
		m.Assoc(entry.Key(), entry.Val())
	}
	return m
}

func (m *TransientMap) Persistent() IPersistentCollection {
	m.ensureEditable()
	m.edit.persisted = true
	if m.array != nil {
		keyVals := append([]any(nil), m.array...)
		return NewMapUniqueKeys(keyVals...)
	}
	return &PersistentHashMap{
		count: m.count,
		root:  m.root,
	}
}

func (m *TransientMap) Count() int {
	m.ensureEditable()
	return m.count
}

func (m *TransientMap) xxx_counted() {}

func (m *TransientMap) ValAt(key any) any {
	return m.ValAtDefault(key, nil)
}

func (m *TransientMap) ValAtDefault(key, fallback any) any {
	m.ensureEditable()
	if m.array != nil {
		if index := m.arrayIndex(key); index >= 0 {
			return m.array[index+1]
		}
		return fallback
	}
	if m.root != nil {
		if _, value, found := m.root.find(0, HashEq(key), key); found {
			return value
		}
	}
	return fallback
}

func (m *TransientMap) ContainsKey(key any) bool {
	m.ensureEditable()
	if m.array != nil {
		return m.arrayIndex(key) >= 0
	}
	if m.root != nil {
		_, _, found := m.root.find(0, HashEq(key), key)
		return found
	}
	return false
}

func (m *TransientMap) EntryAt(key any) IMapEntry {
	m.ensureEditable()
	if m.array != nil {
		if index := m.arrayIndex(key); index >= 0 {
			return NewMapEntry(m.array[index], m.array[index+1])
		}
	} else if m.root != nil {
		foundKey, value, found := m.root.find(0, HashEq(key), key)
		if found {
			return NewMapEntry(foundKey, value)
		}
	}
	return nil
}

func (m *TransientMap) Invoke(args ...any) any {
	switch len(args) {
	case 1:
		return m.ValAt(args[0])
	case 2:
		return m.ValAtDefault(args[0], args[1])
	default:
		panic(NewIllegalArgumentError(
			fmt.Sprintf("map apply expects 1 or 2 arguments, got %d", len(args)),
		))
	}
}

func (m *TransientMap) ApplyTo(args ISeq) any {
	return m.Invoke(seqToSlice(args)...)
}

func (m *TransientMap) arrayIndex(key any) int {
	for i := 0; i < len(m.array); i += 2 {
		if equalKey(m.array[i], key) {
			return i
		}
	}
	return -1
}

func (m *TransientMap) promoteArray() {
	keyVals := m.array
	m.array = nil
	m.root = nil
	m.count = 0
	for i := 0; i < len(keyVals); i += 2 {
		m.assocHAMT(keyVals[i], keyVals[i+1])
	}
}
