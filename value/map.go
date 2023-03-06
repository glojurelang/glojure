package value

import (
	"errors"
	"strings"
)

func SafeMerge(m1, m2 IPersistentMap) IPersistentMap {
	if m1 == nil {
		return m2
	}
	if m2 == nil {
		return m1
	}
	return Merge(m1, m2)
}

func Merge(m1, m2 IPersistentMap) IPersistentMap {
	// TODO: use transient
	var res Associative = m1
	for seq := Seq(m2); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		res = res.Assoc(entry.Key(), entry.Val())
	}
	return res.(IPersistentMap)
}

func mapConj(m IPersistentMap, obj interface{}) Conjer {
	switch obj := obj.(type) {
	case IPersistentVector:
		if obj.Count() != 2 {
			panic(errors.New("Vector argument to map's conj must be a vector with two elements"))
		}
		return m.Assoc(obj.ValAt(0), obj.ValAt(1))
	case IPersistentMap:
		return Merge(m, obj)
	default:
		panic(errors.New("argument to map's conj must be a vector with two elements or a map"))
	}
}

func mapEquals(m IPersistentMap, v2 interface{}) bool {
	if m == v2 {
		return true
	}

	if c, ok := v2.(Counted); ok {
		if m.Count() != c.Count() {
			return false
		}
	}
	assoc, ok := v2.(Associative)
	if !ok {
		return false
	}

	for seq := m.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		if !assoc.ContainsKey(entry.Key()) {
			return false
		}
		if !Equal(entry.Val(), assoc.EntryAt(entry.Key()).Val()) {
			return false
		}
	}

	return true
}

func mapString(m IPersistentMap) string {
	b := strings.Builder{}

	first := true

	// TODO: factor out common namespace
	b.WriteString("{")
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		if !first {
			b.WriteString(", ")
		}
		first = false

		entry := seq.First().(IMapEntry)
		b.WriteString(ToString(entry.Key()))
		b.WriteRune(' ')
		b.WriteString(ToString(entry.Val()))
	}
	b.WriteString("}")
	return b.String()
}
