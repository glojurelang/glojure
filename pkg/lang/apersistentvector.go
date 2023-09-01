package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/murmur3"
)

type (
	APersistentVector interface {
		AFn
		IPersistentVector
		IHashEq
	}

	apvSeq struct {
		v IPersistentVector
		i int
	}
)

func apersistentVectorString(a APersistentVector) string {
	return PrintString(a)
}

func apersistentVectorSeq(a APersistentVector) ISeq {
	if a.Count() > 0 {
		return newAPVSeq(a, 0)
	}
	return nil
}

func apersistentVectorRSeq(a APersistentVector) ISeq {
	if a.Count() > 0 {
		return newAPVRSeq(a, a.Count()-1)
	}
	return nil
}

func apersistentVectorAssoc(a APersistentVector, key, val any) Associative {
	if !IsInteger(key) {
		panic(NewIllegalArgumentError("key must be integer"))
	}

	i := MustAsInt(key)
	return a.AssocN(i, val)
}

func apersistentVectorContainsKey(a APersistentVector, key any) bool {
	if !IsInteger(key) {
		return false
	}

	i := MustAsInt(key)
	return i >= 0 && i < a.Count()
}

func apersistentVectorEntryAt(a APersistentVector, key any) IMapEntry {
	if !IsInteger(key) {
		return nil
	}

	i := MustAsInt(key)
	if i >= 0 && i < a.Count() {
		return NewMapEntry(key, a.Nth(i))
	}
	return nil
}

func apersistentVectorValAt(a APersistentVector, key any) any {
	return apersistentVectorValAtDefault(a, key, nil)
}

func apersistentVectorValAtDefault(a APersistentVector, key, notFound any) any {
	if IsInteger(key) {
		i := MustAsInt(key)
		if i >= 0 && i < a.Count() {
			return a.Nth(i)
		}
	}
	return notFound
}

func apersistentVectorEquiv(a APersistentVector, o any) bool {
	if a == o {
		return true
	}

	switch o := o.(type) {
	case IPersistentVector:
		if o.Count() != a.Count() {
			return false
		}
		for i := 0; i < a.Count(); i++ {
			if !Equiv(a.Nth(i), o.Nth(i)) {
				return false
			}
		}
		return true
	case Sequential:
		ms := Seq(o)
		for i := 0; i < a.Count(); i++ {
			if ms == nil || !Equiv(a.Nth(i), ms.First()) {
				return false
			}
			ms = ms.Next()
		}
		return ms == nil
	default:
		v := reflect.ValueOf(o)
		if !(v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
			return false
		}
		if v.Len() != a.Count() {
			return false
		}
		for i := 0; i < a.Count(); i++ {
			if !Equiv(a.Nth(i), v.Index(i).Interface()) {
				return false
			}
		}
		return true
	}
}

func apersistentVectorHashEq(hc *uint32, a APersistentVector) uint32 {
	if *hc != 0 {
		return *hc
	}
	var n int
	var hash uint32 = 1
	for ; n < a.Count(); n++ {
		hash = 31*hash + HashEq(a.Nth(n))
	}
	hash = murmur3.MixCollHash(hash, uint32(n))
	*hc = hash
	return hash
}

func apersistentVectorInvoke(a APersistentVector, args ...any) any {
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf("vector apply expects one argument, got %d", len(args))))
	}
	if !IsInteger(args[0]) {
		panic(NewIllegalArgumentError("key must be integer"))
	}
	return a.Nth(MustAsInt(args[0]))
}

func apersistentVectorLength(a APersistentVector) int {
	return a.Count()
}

func apersistentVectorPeek(a APersistentVector) any {
	if a.Count() > 0 {
		return a.Nth(a.Count() - 1)
	}
	return nil
}
