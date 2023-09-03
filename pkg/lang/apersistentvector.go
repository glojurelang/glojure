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
		Reversible
	}

	apvSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		v IPersistentVector
		i int
	}

	apvRSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		v IPersistentVector
		i int
	}
)

var (
	_ ASeq       = (*apvSeq)(nil)
	_ IndexedSeq = (*apvSeq)(nil)
	_ IReduce    = (*apvSeq)(nil)

	_ ASeq       = (*apvRSeq)(nil)
	_ IndexedSeq = (*apvRSeq)(nil)
	_ Counted    = (*apvRSeq)(nil)
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

func apersistentVectorEquals(a APersistentVector, o any) bool {
	if a == o {
		return true
	}

	switch o := o.(type) {
	case IPersistentVector:
		if o.Count() != a.Count() {
			return false
		}
		for i := 0; i < a.Count(); i++ {
			if !Equals(a.Nth(i), o.Nth(i)) {
				return false
			}
		}
		return true
	case Sequential:
		ms := Seq(o)
		for i := 0; i < a.Count(); i++ {
			if ms == nil || !Equals(a.Nth(i), ms.First()) {
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
			if !Equals(a.Nth(i), v.Index(i).Interface()) {
				return false
			}
		}
		return true
	}
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

////////////////////////////////////////////////////////////////////////////////

func newAPVSeq(v IPersistentVector, i int) *apvSeq {
	return &apvSeq{
		v: v,
		i: i,
	}
}

func (s *apvSeq) First() any {
	return s.v.Nth(s.i)
}

func (s *apvSeq) Next() ISeq {
	if s.i+1 >= s.v.Count() {
		return nil
	}
	return newAPVSeq(s.v, s.i+1)
}

func (s *apvSeq) Index() int {
	return s.i
}

func (s *apvSeq) Count() int {
	return s.v.Count() - s.i
}

func (s *apvSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *apvSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	return newAPVSeq(s.v, s.i).WithMeta(meta)
}

func (s *apvSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *apvSeq) Reduce(f IFn) any {
	ret := s.v.Nth(s.i)
	for x := s.i + 1; x < s.v.Count(); x++ {
		ret = f.Invoke(ret, s.v.Nth(x))
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *apvSeq) ReduceInit(f IFn, init any) any {
	ret := init
	for x := s.i; x < s.v.Count(); x++ {
		ret = f.Invoke(ret, s.v.Nth(x))
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (s *apvSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *apvSeq) String() string {
	return aseqString(s)
}

func (s *apvSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *apvSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *apvSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *apvSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *apvSeq) More() ISeq {
	return aseqMore(s)
}

func (s *apvSeq) Seq() ISeq {
	return s
}

func (s *apvSeq) xxx_sequential() {}

////////////////////////////////////////////////////////////////////////////////

func newAPVRSeq(v IPersistentVector, i int) *apvRSeq {
	return &apvRSeq{
		v: v,
		i: i,
	}
}

func (s *apvRSeq) First() any {
	return s.v.Nth(s.i)
}

func (s *apvRSeq) Next() ISeq {
	if s.i <= 0 {
		return nil
	}
	return newAPVSeq(s.v, s.i-1)
}

func (s *apvRSeq) Index() int {
	return s.i
}

func (s *apvRSeq) Count() int {
	return s.i + 1
}

func (s *apvRSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *apvRSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	return newAPVRSeq(s.v, s.i).WithMeta(meta)
}

func (s *apvRSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *apvRSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *apvRSeq) String() string {
	return aseqString(s)
}

func (s *apvRSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *apvRSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *apvRSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *apvRSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *apvRSeq) More() ISeq {
	return aseqMore(s)
}

func (s *apvRSeq) Seq() ISeq {
	return s
}

func (s *apvRSeq) xxx_sequential() {}
