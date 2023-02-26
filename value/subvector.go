package value

import "fmt"

type SubVector struct {
	v          IPersistentVector
	start, end int
	meta       IPersistentMap
}

var (
	_ IObj              = (*SubVector)(nil)
	_ IPersistentVector = (*SubVector)(nil)
	// TODO: AFn
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
		return v.Cons(val)
	}
	return NewSubVector(v.meta, v.v.AssocN(v.start+i, val), v.start, v.end)
}

func (v *SubVector) Conj(val interface{}) Conjer {
	return v.Cons(val).(Conjer)
}

func (v *SubVector) Cons(val interface{}) IPersistentVector {
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
		val, _ := v.v.Nth(v.start + kInt)
		return &MapEntry{key: kInt, val: val}
	}
	return nil
}

func (v *SubVector) ValueAt(i int) interface{} {
	val, ok := v.Nth(i)
	if !ok {
		panic("index out of range")
	}
	return val
}

func (v *SubVector) Equal(v2 interface{}) bool {
	other, ok := v2.(IPersistentVector)
	if !ok {
		return false
	}
	if v.Count() != other.Count() {
		return false
	}
	for i := 0; i < v.Count(); i++ {
		vVal, oVal := v.EntryAt(i), other.EntryAt(i)
		if vVal == nil || oVal == nil {
			return vVal == oVal
		}
		if !Equal(vVal, oVal) {
			return false
		}
	}
	return true
}

func (v *SubVector) Nth(i int) (val interface{}, ok bool) {
	return v.v.Nth(v.start + i)
}

func (v *SubVector) NthDefault(i int, def interface{}) interface{} {
	val, ok := v.Nth(i)
	if !ok {
		return def
	}
	return val
}

func (v *SubVector) Peek() interface{} {
	if v.Count() == 0 {
		return nil
	}
	return v.ValueAt(v.Count() - 1)
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
	if v.Count() == 0 {
		return nil
	}
	return NewVectorIterator(v, v.Count()-1, -1)
}

func (v *SubVector) Seq() ISeq {
	if v.Count() == 0 {
		return nil
	}
	return NewVectorIterator(v, 0, 1)
}
