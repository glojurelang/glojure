package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

// Vector is a vector of values.
type (
	Vector struct {
		meta         IPersistentMap
		hash, hasheq uint32

		vec vector.Vector
	}

	PersistentVector = Vector

	TransientVector struct {
		vec *vector.Transient
	}
)

var (
	emptyVector = NewVector()

	_ APersistentVector   = (*Vector)(nil)
	_ IObj                = (*Vector)(nil)
	_ IReduce             = (*Vector)(nil)
	_ IReduceInit         = (*Vector)(nil)
	_ IDrop               = (*Vector)(nil)
	_ IKVReduce           = (*Vector)(nil)
	_ IEditableCollection = (*Vector)(nil)

	_ ITransientVector      = (*TransientVector)(nil)
	_ AFn                   = (*TransientVector)(nil)
	_ ITransientAssociative = (*TransientVector)(nil)
)

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

func (v *Vector) xxx_counted() {}

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
	if i < 0 || i > v.Count() {
		panic(NewIndexOutOfBoundsError())
	}
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
		panic(NewIllegalArgumentError(fmt.Sprintf("vector assoc expects an int as a key, got %T", key)))
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
		panic(NewIllegalArgumentError(fmt.Sprintf("vector apply expects 1 argument, got %d", len(args))))
	}

	i, ok := AsInt(args[0])
	if !ok {
		panic(NewIllegalArgumentError("vector apply takes an int as an argument"))
	}

	if i < 0 || i >= v.Count() {
		panic(NewIllegalArgumentError("index out of bounds"))
	}

	return v.ValAt(i)
}

func (v *Vector) ApplyTo(args ISeq) any {
	return v.Invoke(seqToSlice(args)...)
}

func (v *Vector) Seq() ISeq {
	// TODO: more efficient implementation using vector iterator
	return apersistentVectorSeq(v)
}

func (v *Vector) RSeq() ISeq {
	return apersistentVectorRSeq(v)
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
	if v.meta == meta {
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
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
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
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (v *Vector) KVReduce(f IFn, init any) any {
	for i := 0; i < v.Count(); i++ {
		init = f.Invoke(init, i, v.ValAt(i))
		if IsReduced(init) {
			return init.(IDeref).Deref()
		}
	}
	return init
}

func (v *Vector) Drop(n int) Sequential {
	if n <= 0 {
		return v
	}
	if n >= v.Count() {
		return nil
	}
	return &Vector{
		vec: v.vec.SubVector(n, v.Count()),
	}
}

func (v *Vector) AsTransient() ITransientCollection {
	return &TransientVector{
		vec: vector.NewTransient(v.vec),
	}
}

func (v *Vector) Compare(other any) int {
	otherVec, ok := other.(IPersistentVector)
	if !ok {
		panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare Vector with %T", other)))
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

////////////////////////////////////////////////////////////////////////////////
// TransientVector

func (t *TransientVector) Conj(o any) Conjer {
	t.vec = t.vec.Conj(o)
	return t
}

func (t *TransientVector) ValAt(i any) any {
	return t.ValAtDefault(i, nil)
}

func (t *TransientVector) ValAtDefault(k, def any) any {
	if i, ok := AsInt(k); ok {
		return t.NthDefault(i, def)
	}
	return def
}

func (t *TransientVector) Persistent() IPersistentCollection {
	return &Vector{
		vec: t.vec.Persistent(),
	}
}

func (t *TransientVector) Count() int {
	return t.vec.Count()
}

func (t *TransientVector) xxx_counted() {}

func (t *TransientVector) Nth(i int) any {
	res, ok := t.vec.Index(i)
	if !ok {
		panic(NewIndexOutOfBoundsError())
	}
	return res
}

func (t *TransientVector) NthDefault(i int, def any) any {
	if i >= 0 && i < t.Count() {
		return t.Nth(i)
	}
	return def
}

func (t *TransientVector) AssocN(i int, val any) ITransientVector {
	if i < 0 || i > t.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	t.vec.Assoc(i, val)
	return t
}

func (t *TransientVector) Assoc(key, val any) ITransientAssociative {
	kInt, ok := AsInt(key)
	if !ok {
		panic(NewIllegalArgumentError(fmt.Sprintf("vector assoc expects an int as a key, got %T", key)))
	}
	if kInt < 0 || kInt > t.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	return t.AssocN(kInt, val)
}

func (t *TransientVector) Pop() ITransientVector {
	t.vec = t.vec.Pop()
	return t
}

func (t *TransientVector) ApplyTo(args ISeq) any {
	return t.Invoke(seqToSlice(args)...)
}

func (t *TransientVector) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf("vector apply expects 1 argument, got %d", len(args))))
	}

	i, ok := AsInt(args[0])
	if !ok {
		panic(NewIllegalArgumentError("vector apply takes an int as an argument"))
	}

	if i < 0 || i >= t.Count() {
		panic(NewIllegalArgumentError("index out of bounds"))
	}

	return t.ValAt(i)
}
