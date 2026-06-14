package lang

import (
	"errors"

	"github.com/glojurelang/glojure/internal/murmur3"
)

type (
	APersistentMap interface {
		AFn
		IPersistentMap
		IHashEq
		Hasher
	}
)

func apersistentmapString(a APersistentMap) string {
	return PrintString(a)
}

func apersistentmapAssocEx(a APersistentMap, k, v any) IPersistentMap {
	if a.ContainsKey(k) {
		panic(errors.New("key already present"))
	}
	return a.Assoc(k, v).(IPersistentMap)
}

func apersistentmapCons(a APersistentMap, x any) Conser {
	switch x := x.(type) {
	case IMapEntry:
		return a.Assoc(x.Key(), x.Val()).(Conser)
	case IPersistentVector:
		if x.Count() != 2 {
			panic("vector arg to map conj must be a pair")
		}
		return a.Assoc(MustNth(x, 0), MustNth(x, 1)).(Conser)
	}

	var ret Conser = a
	for seq := Seq(x); seq != nil; seq = seq.Next() {
		ret = ret.Cons(seq.First().(IMapEntry))
	}
	return ret
}

func apersistentmapContainsKey(a APersistentMap, key any) bool {
	return a.EntryAt(key) != nil
}

func apersistentmapEquiv(a APersistentMap, obj any) bool {
	if a == obj {
		return true
	}

	if c, ok := obj.(Counted); ok {
		if a.Count() != c.Count() {
			return false
		}
	}
	assoc, ok := obj.(Associative)
	if !ok {
		return false
	}

	for s := a.Seq(); s != nil; s = s.Next() {
		entry := s.First().(IMapEntry)
		if !assoc.ContainsKey(entry.Key()) {
			return false
		}
		if !Equiv(entry.Val(), assoc.EntryAt(entry.Key()).Val()) {
			return false
		}
	}

	return true
}

func apersistentmapHash(hc *uint32, a APersistentMap) uint32 {
	if *hc != 0 {
		return *hc
	}
	// Following Clojure's APersistentMap.mapHash logic:
	// Sum of (key.hashCode() XOR value.hashCode()) for each entry
	var hash uint32 = 0
	for seq := a.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		keyHash := Hash(entry.Key())
		valHash := Hash(entry.Val())
		hash += keyHash ^ valHash
	}
	*hc = hash
	return hash
}

func apersistentmapHashEq(hc *uint32, a APersistentMap) uint32 {
	if *hc != 0 {
		return *hc
	}
	hash := murmur3.HashUnordered(seqToInternalSeq(a.Seq()), HashEq)
	*hc = hash
	return hash
}

func apersistentmapInvoke(a APersistentMap, args ...any) any {
	if len(args) == 1 {
		return a.ValAt(args[0])
	}
	if len(args) == 2 {
		return a.ValAtDefault(args[0], args[1])
	}
	panic(NewIllegalArgumentError("map expects either 1 or 2 arguments"))
}
