package value

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/persistent/vector"
)

// Vector is a vector of values.
type Vector struct {
	Section
	meta IPersistentMap
	vec  vector.Vector
}

func NewVector(values ...interface{}) *Vector {
	vals := make([]interface{}, len(values))
	for i, v := range values {
		vals[i] = v
	}
	vec := vector.New(vals...)

	return &Vector{
		vec: vec,
	}
}

var (
	_ IPersistentVector = (*Vector)(nil)
)

func (v *Vector) Count() int {
	return v.vec.Len()
}

func (v *Vector) Length() int {
	return v.Count()
}

func (v *Vector) Conj(x interface{}) Conjer {
	return &Vector{
		meta: v.meta,
		vec:  v.vec.Conj(x),
	}
}

func (v *Vector) Cons(item interface{}) IPersistentVector {
	return v.Conj(item).(IPersistentVector)
}

func (v *Vector) AssocN(i int, val interface{}) IPersistentVector {
	return &Vector{vec: v.vec.Assoc(i, val)}
}

func (v *Vector) ContainsKey(key interface{}) bool {
	kInt, ok := AsInt(key)
	if !ok {
		return false
	}
	return kInt >= 0 && kInt < v.Count()
}

func (v *Vector) Assoc(key, val interface{}) Associative {
	kInt, ok := AsInt(key)
	if !ok {
		panic(fmt.Errorf("vector assoc expects an int as a key, got %T", key))
	}
	return v.AssocN(kInt, val)
}

func (v *Vector) EntryAt(key interface{}) IMapEntry {
	kInt, ok := AsInt(key)
	if !ok {
		return nil
	}
	val, ok := v.Nth(kInt)
	if !ok {
		return nil
	}
	return &MapEntry{
		key: key,
		val: val,
	}
}

func (v *Vector) ValueAt(i int) interface{} {
	val, ok := v.vec.Index(i)
	if !ok {
		panic("index out of range")
	}
	if val == nil {
		return nil
	}
	return val
}

func (v *Vector) Nth(i int) (val interface{}, ok bool) {
	if i < 0 || i >= v.Count() {
		return nil, false
	}
	return v.ValueAt(i), true
}

func (v *Vector) NthDefault(i int, def interface{}) interface{} {
	val, ok := v.Nth(i)
	if !ok {
		return def
	}
	return val
}

func (v *Vector) SubVector(start, end int) *Vector {
	return &Vector{vec: v.vec.SubVector(start, end)}
}

func (v *Vector) Enumerate() (<-chan interface{}, func()) {
	rest := v.vec
	return enumerateFunc(func() (interface{}, bool) {
		if rest.Len() == 0 {
			return nil, false
		}
		val, _ := rest.Index(0)
		rest = rest.SubVector(1, rest.Len())
		return val.(interface{}), true
	})
}

func (v *Vector) String() string {
	b := strings.Builder{}

	b.WriteString("[")
	for i := 0; i < v.Count(); i++ {
		el := v.ValueAt(i)
		if el == nil {
			b.WriteString("nil")
		} else {
			b.WriteString(ToString(el))
		}
		if i < v.Count()-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("]")
	return b.String()
}

func (v *Vector) Equal(v2 interface{}) bool {
	other, ok := v2.(*Vector)
	if !ok {
		return false
	}
	if v.Count() != other.Count() {
		return false
	}
	for i := 0; i < v.Count(); i++ {
		vVal, oVal := v.ValueAt(i), other.ValueAt(i)
		if vVal == nil || oVal == nil {
			return vVal == oVal
		}
		if !Equal(vVal, oVal) {
			return false
		}
	}
	return true
}

func (v *Vector) Apply(env Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector apply expects 1 argument, got %d", len(args))
	}

	i, ok := AsInt(args[0])
	if !ok {
		return nil, fmt.Errorf("vector apply takes an int as an argument")
	}

	if i < 0 || i >= v.Count() {
		return nil, fmt.Errorf("index out of bounds")
	}

	return v.ValueAt(i), nil
}

func (v *Vector) Seq() ISeq {
	return NewVectorIterator(v, 0, 1)
}

func (v *Vector) RSeq() ISeq {
	return NewVectorIterator(v, v.Count()-1, -1)
}

func (v *Vector) Peek() interface{} {
	if v.Count() == 0 {
		return nil
	}
	return v.ValueAt(v.Count() - 1)
}

func (v *Vector) Pop() IPersistentStack {
	if v.Count() == 0 {
		panic("can't pop an empty vector")
	}
	return v.SubVector(0, v.Count()-1)
}

func (v *Vector) GoValue() interface{} {
	vals := make([]interface{}, v.Count())
	for i := 0; i < v.Count(); i++ {
		vals[i] = v.ValueAt(i)
	}
	return vals
}

func (v *Vector) Meta() IPersistentMap {
	return v.meta
}

func (v *Vector) WithMeta(meta IPersistentMap) interface{} {
	if Equal(v.meta, meta) {
		return v
	}

	cpy := *v
	cpy.meta = meta
	return &cpy
}
