package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

var (
	emptyVector = NewVector()
)

// Vector is a vector of values.
type Vector struct {
	meta         IPersistentMap
	hash, hasheq uint32

	vec vector.Vector
}

type PersistentVector = Vector

func NewVector(values ...any) *Vector {
	vals := make([]any, len(values))
	for i, v := range values {
		vals[i] = v
	}
	vec := vector.New(vals...)

	return &Vector{
		vec: vec,
	}
}

func NewVectorFromCollection(c any) *Vector {
	// TODO: match clojure's behavior here. for now, just make it work
	// for seqs.
	var items []any
	for seq := Seq(c); seq != nil; seq = seq.Next() {
		items = append(items, seq.First())
	}
	return NewVector(items...)
}

func NewLazilyPersistentVector(x any) IPersistentVector {
	// TODO: IReduceInit, Iterable
	switch x := x.(type) {
	case ISeq:
		return NewVectorFromCollection(x)
	default:
		return NewVector(toSlice(x)...)
	}
}

var (
	_ APersistentVector = (*Vector)(nil)
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

func (v *Vector) Cons(x any) Conser {
	return &Vector{
		meta: v.meta,
		vec:  v.vec.Conj(x),
	}
}

func (v *Vector) AssocN(i int, val any) IPersistentVector {
	return &Vector{vec: v.vec.Assoc(i, val)}
}

func (v *Vector) ContainsKey(key any) bool {
	kInt, ok := AsInt(key)
	if !ok {
		return false
	}
	return kInt >= 0 && kInt < v.Count()
}

func (v *Vector) Assoc(key, val any) Associative {
	kInt, ok := AsInt(key)
	if !ok {
		panic(fmt.Errorf("vector assoc expects an int as a key, got %T", key))
	}
	return v.AssocN(kInt, val)
}

func (v *Vector) EntryAt(key any) IMapEntry {
	kInt, ok := AsInt(key)
	if !ok {
		return nil
	}
	val := v.NthDefault(kInt, notFound)
	if val == notFound {
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

func (v *Vector) ValAt(i any) any {
	return v.ValAtDefault(i, nil)
}

func (v *Vector) ValAtDefault(k, def any) any {
	if i, ok := AsInt(k); ok {
		return v.NthDefault(i, def)
	}
	return def
}

func (v *Vector) Nth(i int) any {
	res, ok := v.vec.Index(i)
	if !ok {
		panic(NewIndexOutOfBoundsError())
	}
	return res
}

func (v *Vector) NthDefault(i int, def any) any {
	if i >= 0 && i < v.Count() {
		return v.Nth(i)
	}
	return def
}

func (v *Vector) String() string {
	return apersistentVectorString(v)
}

func (v *Vector) Equals(v2 any) bool {
	return apersistentVectorEquals(v, v2)
}

func (v *Vector) Equiv(v2 any) bool {
	return apersistentVectorEquiv(v, v2)
}

func (v *Vector) Invoke(args ...any) any {
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

func (v *Vector) ApplyTo(args ISeq) any {
	return v.Invoke(seqToSlice(args)...)
}

func (v *Vector) Seq() ISeq {
	if v.Count() == 0 {
		return nil
	}
	// TODO: chunked seq
	return NewVectorIterator(v, 0, 1)
}

func (v *Vector) RSeq() ISeq {
	if v.Count() == 0 {
		return nil
	}
	return NewVectorIterator(v, v.Count()-1, -1)
}

func (v *Vector) Peek() any {
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

func (v *Vector) WithMeta(meta IPersistentMap) any {
	if Equal(v.meta, meta) {
		return v
	}

	cpy := *v
	cpy.meta = meta
	return &cpy
}

func (v *Vector) HashEq() uint32 {
	return apersistentVectorHashEq(&v.hasheq, v)
}

func (v *Vector) ReduceInit(f IFn, init any) any {
	res := init
	for i := 0; i < v.Count(); i++ {
		res = f.Invoke(res, v.ValAt(i))
	}
	return res
}

func (v *Vector) Reduce(f IFn) any {
	if v.Count() == 0 {
		return f.Invoke()
	}
	res := v.ValAt(0)
	for i := 1; i < v.Count(); i++ {
		res = f.Invoke(res, v.ValAt(i))
	}
	return res
}

func toSlice(x any) []any {
	if x == nil {
		return nil
	}

	val := reflect.ValueOf(x)
	if val.Type().Kind() == reflect.Slice {
		res := make([]any, val.Len())
		for i := 0; i < len(res); i++ {
			res[i] = val.Index(i).Interface()
		}
		return res
	}

	if idxd, ok := x.(Indexed); ok {
		count := Count(x)
		res := make([]any, count)
		for i := 0; i < count; i++ {
			res = append(res, idxd.Nth(i))
		}
		return res
	}

	panic(fmt.Sprintf("unable to convert %T to slice", x))
}
