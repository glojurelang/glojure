package lang

import "fmt"

type SubVector struct {
	meta         IPersistentMap
	hash, hasheq uint32

	v          IPersistentVector
	start, end int
}

var (
	_ APersistentVector = (*SubVector)(nil)
	_ IObj              = (*SubVector)(nil)
	_ IPersistentVector = (*SubVector)(nil)
)

func NewSubVector(meta IPersistentMap, v IPersistentVector, start, end int) *SubVector {
	return &SubVector{
		v:     v,
		start: start,
		end:   end,
		meta:  meta,
	}
}

func (v *SubVector) xxx_sequential() {}

func (v *SubVector) Meta() IPersistentMap {
	return v.meta
}

func (v *SubVector) WithMeta(meta IPersistentMap) interface{} {
	if meta == v.meta {
		return v
	}
	return NewSubVector(meta, v.v, v.start, v.end)
}

func (v *SubVector) Assoc(key, val interface{}) Associative {
	kInt, ok := AsInt(key)
	if !ok {
		panic(fmt.Errorf("vector assoc expects an int as a key, got %T", key))
	}
	return v.AssocN(kInt, val)
}

func (v *SubVector) AssocN(i int, val interface{}) IPersistentVector {
	if v.start+i > v.end {
		panic(fmt.Errorf("index out of bounds: %d", i))
	}
	if v.start+i == v.end {
		return v.Cons(val).(IPersistentVector)
	}
	return NewSubVector(v.meta, v.v.AssocN(v.start+i, val), v.start, v.end)
}

func (v *SubVector) Cons(val interface{}) Conser {
	return NewSubVector(v.meta, v.v.AssocN(v.end, val), v.start, v.end+1)
}

func (v *SubVector) ContainsKey(key interface{}) bool {
	kInt, ok := AsInt(key)
	if !ok {
		return false
	}
	return kInt >= 0 && kInt < v.end-v.start
}

func (v *SubVector) Count() int {
	return v.end - v.start
}

func (v *SubVector) Length() int {
	return v.Count()
}

func (v *SubVector) EntryAt(key interface{}) IMapEntry {
	kInt, ok := AsInt(key)
	if !ok {
		return nil
	}
	if kInt >= 0 && kInt < v.end-v.start {
		val := v.v.Nth(v.start + kInt)
		return &MapEntry{key: kInt, val: val}
	}
	return nil
}

func (v *SubVector) ValAt(i interface{}) interface{} {
	return v.ValAtDefault(i, nil)
}

func (v *SubVector) ValAtDefault(k, def interface{}) interface{} {
	if i, ok := AsInt(k); ok {
		return v.NthDefault(i, def)
	}
	return def
}

func (v *SubVector) Equals(v2 interface{}) bool {
	return apersistentVectorEquals(v, v2)
}

func (v *SubVector) Equiv(v2 interface{}) bool {
	return apersistentVectorEquiv(v, v2)
}

func (v *SubVector) Nth(i int) interface{} {
	if v.start+i >= v.end || i < 0 {
		panic(NewIndexOutOfBoundsError())
	}
	return v.v.Nth(v.start + i)
}

func (v *SubVector) NthDefault(i int, def interface{}) interface{} {
	if i >= 0 && i < v.Count() {
		return v.Nth(i)
	}
	return def
}

func (v *SubVector) Peek() interface{} {
	if v.Count() == 0 {
		return nil
	}
	return v.ValAt(v.Count() - 1)
}

func (v *SubVector) Pop() IPersistentStack {
	if v.Count() == 0 {
		panic("can't pop an empty vector")
	}
	if v.end-v.start == 1 {
		return emptyVector
	}
	return NewSubVector(nil, v, 0, v.Count()-1)
}

func (v *SubVector) RSeq() ISeq {
	return apersistentVectorRSeq(v)
}

func (v *SubVector) Seq() ISeq {
	return apersistentVectorSeq(v)
}

func (v *SubVector) IsEmpty() bool {
	return v.Count() == 0
}

func (v *SubVector) Empty() IPersistentCollection {
	return emptyVector.WithMeta(v.meta).(IPersistentCollection)
}

func (v *SubVector) ApplyTo(args ISeq) any {
	return afnApplyTo(v, args)
}

func (v *SubVector) Invoke(args ...any) any {
	return apersistentVectorInvoke(v, args)
}

func (v *SubVector) HashEq() uint32 {
	return apersistentVectorHashEq(&v.hasheq, v)
}

func (v *SubVector) Compare(other any) int {
	otherVec, ok := other.(IPersistentVector)
	if !ok {
		panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare SubVector with %T", other)))
	}

	myCount := v.Count()
	otherCount := otherVec.Count()

	// Compare lengths first
	if myCount < otherCount {
		return -1
	} else if myCount > otherCount {
		return 1
	}

	// Compare element by element
	for i := 0; i < myCount; i++ {
		cmp := Compare(v.Nth(i), otherVec.Nth(i))
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}
