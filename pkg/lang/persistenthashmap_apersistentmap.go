// GENERATED CODE. DO NOT EDIT
package lang

import (
	"errors"
	"fmt"
)

func (m *PersistentHashMap) Conj(x interface{}) Conjer {
	switch x := x.(type) {
	case *MapEntry:
		return m.Assoc(x.Key(), x.Val()).(Conjer)
	case IPersistentVector:
		if x.Count() != 2 {
			panic("vector arg to map conj must be a pair")
		}
		return m.Assoc(MustNth(x, 0), MustNth(x, 1)).(Conjer)
	}

	var ret Conjer = m
	for seq := Seq(x); seq != nil; seq = seq.Next() {
		ret = ret.Conj(seq.First().(*MapEntry))
	}
	return ret
}

func (m *PersistentHashMap) ContainsKey(key interface{}) bool {
	return m.EntryAt(key) != nil
}

func (m *PersistentHashMap) AssocEx(k, v interface{}) IPersistentMap {
	if m.ContainsKey(k) {
		panic(errors.New("key already present"))
	}
	return m.Assoc(k, v).(IPersistentMap)
}

func (m *PersistentHashMap) Equal(v2 interface{}) bool {
	return mapEquals(m, v2)
}

func (m *PersistentHashMap) IsEmpty() bool {
	return m.Count() == 0
}

func (m *PersistentHashMap) ValAt(key interface{}) interface{} {
	return m.ValAtDefault(key, nil)
}

// IFn methods

func (m *PersistentHashMap) Invoke(args ...interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("map apply expects 1 argument, got %d", len(args)))
	}

	return m.ValAt(args[0])
}

func (m *PersistentHashMap) ApplyTo(args ISeq) interface{} {
	return m.Invoke(seqToSlice(args)...)
}
