package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

const vectorInlineSize = 4

// Vector is a vector of values.
type (
	Vector struct {
		meta         IPersistentMap
		hash, hasheq uint32

		vec    vector.Vector
		inline []any
	}

	// inlineVectorStorage keeps small vectors to one allocation without making
	// every large Vector carry unused inline capacity. A pointer to Vector keeps
	// its owner alive.
	inlineVectorStorage struct {
		Vector
		values [vectorInlineSize]any
	}

	PersistentVector = Vector

	TransientVector struct {
		vec *vector.Transient
	}
)

var (
	emptyVector = &Vector{}

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
	if len(values) == 0 {
		return emptyVector
	}
	if len(values) <= vectorInlineSize {
		return newInlineVector(nil, values)
	}
	return &Vector{
		vec: vector.New(values...),
	}
}

func newInlineVector(meta IPersistentMap, values []any) *Vector {
	storage := &inlineVectorStorage{}
	storage.Vector.meta = meta
	storage.Vector.inline = storage.values[:len(values)]
	copy(storage.Vector.inline, values)
	return &storage.Vector
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
	if v.vec == nil {
		return len(v.inline)
	}
	return v.vec.Len()
}

func (v *Vector) xxx_counted() {}

func (v *Vector) Length() int {
	return v.Count()
}

func (v *Vector) Cons(x any) Conser {
	if v.vec == nil {
		if len(v.inline) < vectorInlineSize {
			result := newInlineVector(v.meta, v.inline)
			result.inline = result.inline[:len(v.inline)+1]
			result.inline[len(v.inline)] = x
			return result
		}
		values := make([]any, vectorInlineSize+1)
		copy(values, v.inline)
		values[vectorInlineSize] = x
		return &Vector{
			meta: v.meta,
			vec:  vector.New(values...),
		}
	}
	return &Vector{
		meta: v.meta,
		vec:  v.vec.Conj(x),
	}
}

func (v *Vector) AssocN(i int, val any) IPersistentVector {
	if i < 0 || i > v.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	if v.vec == nil {
		if i == len(v.inline) {
			return v.Cons(val).(IPersistentVector)
		}
		result := newInlineVector(v.meta, v.inline)
		result.inline[i] = val
		return result
	}
	return &Vector{meta: v.meta, vec: v.vec.Assoc(i, val)}
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
	if v.vec == nil {
		if i < 0 || i >= len(v.inline) {
			panic(NewIndexOutOfBoundsError())
		}
		return v.inline[i]
	}
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

// Rseq is an alias for RSeq, needed because FieldOrMethod capitalizes
// only the first letter of "rseq" to get "Rseq", not "RSeq".
func (v *Vector) Rseq() ISeq {
	return v.RSeq()
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
	if v.vec == nil {
		return newInlineVector(v.meta, v.inline[:len(v.inline)-1])
	}
	return &Vector{
		meta: v.meta,
		vec:  v.vec.Pop(),
	}
}

func (v *Vector) Meta() IPersistentMap {
	return v.meta
}

func (v *Vector) WithMeta(meta IPersistentMap) any {
	if v.meta == meta {
		return v
	}

	if v.vec == nil && len(v.inline) != 0 {
		cpy := newInlineVector(meta, v.inline)
		cpy.hash, cpy.hasheq = v.hash, v.hasheq
		return cpy
	}
	cpy := *v
	cpy.meta = meta
	return &cpy
}

func (v *Vector) HashEq() uint32 {
	return apersistentVectorHashEq(&v.hasheq, v)
}

func (v *Vector) Hash() uint32 {
	return apersistentVectorHash(&v.hash, v)
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
	if v.vec == nil {
		result := NewVector(v.inline[n:]...)
		result.meta = v.meta
		return result
	}
	return &Vector{
		vec: v.vec.SubVector(n, v.Count()),
	}
}

func (v *Vector) AsTransient() ITransientCollection {
	vec := v.vec
	if vec == nil {
		vec = vector.New(v.inline...)
	}
	return &TransientVector{
		vec: vector.NewTransient(vec),
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
	vec := t.vec.Persistent()
	if vec.Len() <= vectorInlineSize {
		values := make([]any, vec.Len())
		for i := range values {
			values[i], _ = vec.Index(i)
		}
		return NewVector(values...)
	}
	return &Vector{vec: vec}
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
