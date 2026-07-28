package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

// Vector is a vector of values.
type (
	Vector struct {
		attrs *vectorAttrs
		vec   vector.Persistent
	}

	// vectorAttrs keeps metadata and lazily populated hash caches off the hot
	// path for ordinary vectors, where all three values are absent.
	vectorAttrs struct {
		meta         IPersistentMap
		hash, hasheq uint32
	}

	// vectorUpdateStorage co-allocates a vector version with the immutable tail
	// entry created by Cons.
	vectorUpdateStorage struct {
		Vector
		tail vector.TailStorage
	}

	vectorReplaceStorage struct {
		Vector
		tail vector.ReplaceTailStorage
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
	return &Vector{
		vec: vector.NewPersistent(values...),
	}
}

// CanTransientlyUpdateInt64Vector reports whether v can enter the typed
// transient-vector update fast path. The element check is intentionally a
// guard, not a coercion. Metadata takes the persistent fallback because the
// transient representation does not retain it. The size limit keeps updates
// in the independently copied tail until tree-backed transients use edit-token
// copy-on-write.
func CanTransientlyUpdateInt64Vector(v *Vector) bool {
	if v.Count() > 32 || v.Meta() != nil {
		return false
	}
	for i := 0; i < v.Count(); i++ {
		if _, ok := v.Nth(i).(int64); !ok {
			return false
		}
	}
	return true
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
	storage := &vectorUpdateStorage{}
	storage.attrs = newVectorAttrs(v.Meta())
	storage.vec = v.vec.ConjValueInto(x, &storage.tail)
	return &storage.Vector
}

func (v *Vector) AssocN(i int, val any) IPersistentVector {
	if i < 0 || i > v.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	result, ok := v.vec.AssocValue(i, val)
	if !ok {
		panic(NewIndexOutOfBoundsError())
	}
	return &Vector{attrs: newVectorAttrs(v.Meta()), vec: result}
}

// ReplaceLast returns a persistent vector whose final value is val.
func (v *Vector) ReplaceLast(val any) *Vector {
	storage := &vectorReplaceStorage{}
	result, ok := v.vec.ReplaceLastValueInto(val, &storage.tail)
	if !ok {
		panic("can't pop an empty vector")
	}
	storage.attrs = newVectorAttrs(v.Meta())
	storage.vec = result
	return &storage.Vector
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
	return emptyVector.WithMeta(v.Meta()).(IPersistentCollection)
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
	result, ok := v.vec.PopValue()
	if !ok {
		panic("can't pop an empty vector")
	}
	return &Vector{
		attrs: newVectorAttrs(v.Meta()),
		vec:   result,
	}
}

func (v *Vector) Meta() IPersistentMap {
	if v.attrs == nil {
		return nil
	}
	return v.attrs.meta
}

func (v *Vector) WithMeta(meta IPersistentMap) any {
	if v.Meta() == meta {
		return v
	}

	cpy := *v
	if v.attrs == nil {
		cpy.attrs = newVectorAttrs(meta)
	} else {
		attrs := *v.attrs
		attrs.meta = meta
		cpy.attrs = &attrs
	}
	return &cpy
}

func (v *Vector) HashEq() uint32 {
	return apersistentVectorHashEq(&v.ensureAttrs().hasheq, v)
}

func (v *Vector) Hash() uint32 {
	return apersistentVectorHash(&v.ensureAttrs().hash, v)
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
	return NewSubVector(v.Meta(), v, n, v.Count())
}

func newVectorAttrs(meta IPersistentMap) *vectorAttrs {
	if meta == nil {
		return nil
	}
	return &vectorAttrs{meta: meta}
}

func (v *Vector) ensureAttrs() *vectorAttrs {
	if v.attrs == nil {
		v.attrs = &vectorAttrs{}
	}
	return v.attrs
}

func (v *Vector) AsTransient() ITransientCollection {
	return &TransientVector{
		vec: vector.NewTransient(&v.vec),
	}
}

// AsTransientForUpdate creates the compact transient representation used by
// compiler-proven assoc-heavy update regions.
func (v *Vector) AsTransientForUpdate() *TransientVector {
	return &TransientVector{
		vec: vector.NewUpdateTransient(&v.vec),
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
	if vec.Len() == 0 {
		return emptyVector
	}
	return &Vector{vec: *vec}
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
