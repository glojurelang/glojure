// GENERATED CODE. DO NOT EDIT
package value

import (
	"errors"
	"fmt"
)

func (m *PersistentStructMap) Conj(x interface{}) Conjer {
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

func (m *PersistentStructMap) ContainsKey(key interface{}) bool {
	return m.EntryAt(key) != nil
}

func (m *PersistentStructMap) AssocEx(k, v interface{}) IPersistentMap {
	if m.ContainsKey(k) {
		panic(errors.New("key already present"))
	}
	return m.Assoc(k, v).(IPersistentMap)
}

func (m *PersistentStructMap) Equal(v2 interface{}) bool {
	return mapEquals(m, v2)
}

func (m *PersistentStructMap) IsEmpty() bool {
	return m.Count() == 0
}

func (m *PersistentStructMap) ValAt(key interface{}) interface{} {
	return m.ValAtDefault(key, nil)
}

// IFn methods

func (m *PersistentStructMap) Invoke(args ...interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("map apply expects 1 argument, got %d", len(args)))
	}

	return m.ValAt(args[0])
}

func (m *PersistentStructMap) ApplyTo(args ISeq) interface{} {
	return m.Invoke(seqToSlice(args)...)
}
