package lang

import (
	"reflect"

	"github.com/glojurelang/glojure/internal/murmur3"
)

type (
	ASeq interface {
		IObj
		Hasher
		IHashEq
		Equiver
		Equalser
		Counter
		Seqable
		ISeq
		Sequential

		String() string
		Empty() IPersistentCollection

		// Clojure includes Java List ops. We omit these.
	}
)

func aseqMore(a ASeq) ISeq {
	s := a.Next()
	if s == nil {
		return emptyList
	}
	return s
}

func aseqCount(a ASeq) int {
	i := 1
	for s := a.Next(); s != nil; s, i = s.Next(), i+1 {
		if sc, ok := s.(Counted); ok {
			return i + sc.Count()
		}
	}
	return i
}

func aseqCons(a ASeq, o any) Conser {
	return NewCons(o, a)
}

func aseqEmpty() IPersistentCollection {
	return emptyList
}

func aseqEquiv(a ASeq, obj any) bool {
	if a == obj {
		return true
	}

	_, isSequential := obj.(Sequential)
	objV := reflect.ValueOf(obj)
	if !isSequential && objV.Kind() != reflect.Slice {
		return false
	}

	if ac, ok := a.(Counted); ok {
		if bc, ok := obj.(Counted); ok {
			if ac.Count() != bc.Count() {
				return false
			}
		}
	}

	ms := Seq(obj)
	for s := Seq(a); s != nil; s, ms = s.Next(), ms.Next() {
		if ms == nil || !Equiv(s.First(), ms.First()) {
			return false
		}
	}

	return ms == nil
}

func aseqEquals(a ASeq, obj any) bool {
	return aseqEquiv(a, obj)
}

func aseqHash(hc *uint32, a ASeq) uint32 {
	if *hc != 0 {
		return *hc
	}

	hash := uint32(1)
	for s := a.Seq(); s != nil; s = s.Next() {
		var h uint32
		first := s.First()
		if first != nil {
			h = Hash(first)
		}
		hash = 31*hash + h
	}
	*hc = hash
	return hash
}

func aseqHashEq(hc *uint32, a ASeq) uint32 {
	if *hc != 0 {
		return *hc
	}
	hash := murmur3.HashOrdered(seqToInternalSeq(a), HashEq)
	*hc = hash
	return hash
}

func aseqString(a ASeq) string {
	return PrintString(a)
}
