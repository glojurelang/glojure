package lang

import "reflect"

type (
	GoMapSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		iter *reflect.MapIter
	}
)

var (
	_ ASeq        = (*GoMapSeq)(nil)
	_ IKVReduce   = (*GoMapSeq)(nil)
	_ IReduce     = (*GoMapSeq)(nil)
	_ IReduceInit = (*GoMapSeq)(nil)
)

func NewGoMapSeq(gm any) *GoMapSeq {
	v := reflect.ValueOf(gm)
	if v.Kind() != reflect.Map {
		panic(NewIllegalArgumentError("argument to NewGoMapSeq must be a map"))
	}
	iter := v.MapRange()
	if !iter.Next() {
		return nil
	}
	return &GoMapSeq{
		iter: iter,
	}
}

func (s *GoMapSeq) cloneIter() *reflect.MapIter {
	cpy := *s.iter
	return &cpy
}

func (s *GoMapSeq) xxx_sequential() {}

func (s *GoMapSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *GoMapSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *GoMapSeq) String() string {
	return aseqString(s)
}

func (s *GoMapSeq) Seq() ISeq {
	return s
}

func (s *GoMapSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *GoMapSeq) First() any {
	return mapIterEntry(s.iter)
}

func (s *GoMapSeq) Next() ISeq {
	iter := s.cloneIter()
	if !iter.Next() {
		return nil
	}
	return &GoMapSeq{
		meta: s.meta,
		iter: iter,
	}
}

func (s *GoMapSeq) More() ISeq {
	return aseqMore(s)
}

func (s *GoMapSeq) Count() int {
	return aseqCount(s)
}

func (s *GoMapSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *GoMapSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *GoMapSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *GoMapSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *GoMapSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *GoMapSeq) Reduce(f IFn) any {
	if s.Count() == 0 {
		return f.Invoke()
	}

	iter := s.cloneIter()
	var acc any = mapIterEntry(iter)
	for iter.Next() {
		acc = f.Invoke(acc, mapIterEntry(iter))
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}

func (s *GoMapSeq) ReduceInit(f IFn, init any) any {
	acc := init
	iter := s.cloneIter()
	for iter.Next() {
		acc = f.Invoke(acc, mapIterEntry(iter))
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}

func (s *GoMapSeq) KVReduce(f IFn, init any) any {
	acc := init
	iter := s.cloneIter()
	for iter.Next() {
		acc = f.Invoke(acc, iter.Key().Interface(), iter.Value().Interface())
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}

func mapIterEntry(iter *reflect.MapIter) *MapEntry {
	return NewMapEntry(iter.Key().Interface(), iter.Value().Interface())
}
