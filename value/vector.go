package value

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/persistent/vector"
)

var (
	emptyVector = NewVector()
)

// Vector is a vector of values.
type Vector struct {
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

func NewVectorFromCollection(c interface{}) *Vector {
	// TODO: match clojure's behavior here. for now, just make it work
	// for seqs.
	var items []interface{}
	for seq := Seq(c); seq != nil; seq = seq.Next() {
		items = append(items, seq.First())
	}
	return NewVector(items...)
}

func NewLazilyPersistentVector(x interface{}) IPersistentVector {
	// TODO: IReduceInit, Iterable
	switch x := x.(type) {
	case ISeq:
		return NewVectorFromCollection(x)
	default:
		return NewVector(toSlice(x)...)
	}
}

var (
	_ IPersistentVector = (*Vector)(nil)
	_ IFn               = (*Vector)(nil)
	_ IReduce           = (*Vector)(nil)
	_ IReduceInit       = (*Vector)(nil)
)

func (v *Vector) xxx_sequential() {}

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

func (v *Vector) IsEmpty() bool {
	return v.Count() == 0
}

func (v *Vector) Empty() IPersistentCollection {
	return emptyVector.WithMeta(v.meta).(IPersistentCollection)
}

func (v *Vector) ValAt(i interface{}) interface{} {
	return v.ValAtDefault(i, nil)
}

func (v *Vector) ValAtDefault(k, def interface{}) interface{} {
	if i, ok := AsInt(k); ok {
		if val, ok := v.Nth(i); ok {
			return val
		}
	}
	return def
}

func (v *Vector) Nth(i int) (val interface{}, ok bool) {
	return v.vec.Index(i)
}

func (v *Vector) NthDefault(i int, def interface{}) interface{} {
	val, ok := v.Nth(i)
	if !ok {
		return def
	}
	return val
}

func (v *Vector) String() string {
	return PrintString(v)
}

func (v *Vector) Equal(v2 interface{}) bool {
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
		if !Equal(vVal.Val(), oVal.Val()) {
			return false
		}
	}
	return true
}

func (v *Vector) Invoke(args ...interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("vector apply expects 1 argument, got %d", len(args)))
	}

	i, ok := AsInt(args[0])
	if !ok {
		panic(fmt.Errorf("vector apply takes an int as an argument"))
	}

	if i < 0 || i >= v.Count() {
		panic(fmt.Errorf("index out of bounds"))
	}

	return v.ValAt(i)
}

func (v *Vector) ApplyTo(args ISeq) interface{} {
	return v.Invoke(seqToSlice(args)...)
}

func (v *Vector) Seq() ISeq {
	if v.Count() == 0 {
		return nil
	}
	return NewVectorIterator(v, 0, 1)
}

func (v *Vector) RSeq() ISeq {
	if v.Count() == 0 {
		return nil
	}
	return NewVectorIterator(v, v.Count()-1, -1)
}

func (v *Vector) Peek() interface{} {
	if v.Count() == 0 {
		return nil
	}
	return v.ValAt(v.Count() - 1)
}

func (v *Vector) Pop() IPersistentStack {
	if v.Count() == 0 {
		panic("can't pop an empty vector")
	}
	if v.Count() == 1 {
		return emptyVector
	}
	return NewSubVector(nil, v, 0, v.Count()-1)
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

func (v *Vector) ReduceInit(f IFn, init interface{}) interface{} {
	res := init
	for i := 0; i < v.Count(); i++ {
		res = f.Invoke(res, v.ValAt(i))
	}
	return res
}

func (v *Vector) Reduce(f IFn) interface{} {
	if v.Count() == 0 {
		return f.Invoke()
	}
	res := v.ValAt(0)
	for i := 1; i < v.Count(); i++ {
		res = f.Invoke(res, v.ValAt(i))
	}
	return res
}

func toSlice(x interface{}) []interface{} {
	if x == nil {
		return nil
	}

	val := reflect.ValueOf(x)
	if val.Type().Kind() == reflect.Slice {
		res := make([]interface{}, val.Len())
		for i := 0; i < len(res); i++ {
			res[i] = val.Index(i).Interface()
		}
		return res
	}

	if idxd, ok := x.(Indexed); ok {
		count := Count(x)
		res := make([]interface{}, count)
		for i := 0; i < count; i++ {
			val, _ := idxd.Nth(i)
			res = append(res, val)
		}
		return res
	}

	panic(fmt.Sprintf("unable to convert %T to slice", x))
}
