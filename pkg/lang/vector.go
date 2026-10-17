package lang

import (
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

// Vector is a vector of values.
type (
	Vector struct {
		attrs atomic.Pointer[vectorAttrs]
		vec   vector.Persistent
	}

	// vectorAttrs keeps metadata and lazily populated hash caches off the hot
	// path for ordinary vectors, where all three values are absent.
	vectorAttrs struct {
		meta         IPersistentMap
		hash, hasheq uint32
		// flat holds every element in order for vectors built in one go
		// from a known length, so Nth is a slice index instead of a trie
		// walk. Derived vectors get fresh attrs and never inherit it.
		flat []any
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
	switch len(values) {
	case 0:
		return emptyVector
	case 1:
		s := &struct {
			Vector
			tail  vector.InlineTailStorage
			items [1]any
		}{}
		copy(s.items[:], values)
		s.vec = vector.InitInlineTail(&s.tail, s.items[:])
		return &s.Vector
	case 2:
		s := &struct {
			Vector
			tail  vector.InlineTailStorage
			items [2]any
		}{}
		copy(s.items[:], values)
		s.vec = vector.InitInlineTail(&s.tail, s.items[:])
		return &s.Vector
	case 3:
		s := &struct {
			Vector
			tail  vector.InlineTailStorage
			items [3]any
		}{}
		copy(s.items[:], values)
		s.vec = vector.InitInlineTail(&s.tail, s.items[:])
		return &s.Vector
	case 4:
		s := &struct {
			Vector
			tail  vector.InlineTailStorage
			items [4]any
		}{}
		copy(s.items[:], values)
		s.vec = vector.InitInlineTail(&s.tail, s.items[:])
		return &s.Vector
	}
	return &Vector{
		vec: vector.NewPersistent(values...),
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

func (v *Vector) IndexOf(value any) int {
	for index := 0; index < v.Count(); index++ {
		if Equals(v.Nth(index), value) {
			return index
		}
	}
	return -1
}

func (v *Vector) LastIndexOf(value any) int {
	for index := v.Count() - 1; index >= 0; index-- {
		if Equals(v.Nth(index), value) {
			return index
		}
	}
	return -1
}

func (v *Vector) Cons(x any) Conser {
	storage := &vectorUpdateStorage{}
	if meta := v.Meta(); meta != nil {
		storage.attrs.Store(newVectorAttrs(meta))
	}
	storage.vec = v.vec.ConjValueInto(x, &storage.tail)
	return &storage.Vector
}

func (v *Vector) AssocN(i int, val any) IPersistentVector {
	if i < 0 || i > v.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	if i < v.Count() {
		previous, _ := v.vec.Index(i)
		if sameVectorValue(previous, val) {
			return newVectorWithMeta(v.vec, v.Meta())
		}
	}
	result, ok := v.vec.AssocValue(i, val)
	if !ok {
		panic(NewIndexOutOfBoundsError())
	}
	return newVectorWithMeta(result, v.Meta())
}

// Only identity-preserving replacements can reuse the vector. Equality alone
// would discard a distinct replacement object, and function code pointers do
// not distinguish closures with different captured values.
func sameVectorValue(previous, replacement any) bool {
	switch value := previous.(type) {
	case int:
		other, ok := replacement.(int)
		return ok && value == other
	case nil, bool, int64, string, Char, Keyword,
		*Map, *Vector, *List, *EmptyList, *PersistentHashMap, *Symbol,
		*Var, *LazySeq, *Cons, *Atom, *MetaFn, *MultiArityFn, *Volatile:
		return Identical(previous, replacement)
	default:
		return false
	}
}

// ReplaceLast returns a persistent vector whose final value is val.
func (v *Vector) ReplaceLast(val any) *Vector {
	storage := &vectorReplaceStorage{}
	result, ok := v.vec.ReplaceLastValueInto(val, &storage.tail)
	if !ok {
		panic("can't pop an empty vector")
	}
	if meta := v.Meta(); meta != nil {
		storage.attrs.Store(newVectorAttrs(meta))
	}
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
	if Identical(val, notFound) {
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
	if a := v.attrs.Load(); a != nil && a.flat != nil {
		if i < 0 || i >= len(a.flat) {
			panic(NewIndexOutOfBoundsError())
		}
		return a.flat[i]
	}
	res, ok := v.vec.Index(i)
	if !ok {
		panic(NewIndexOutOfBoundsError())
	}
	return res
}

func (v *Vector) NthDefault(i int, def any) any {
	if a := v.attrs.Load(); a != nil && a.flat != nil {
		if i >= 0 && i < len(a.flat) {
			return a.flat[i]
		}
		return def
	}
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
	return newVectorSeq(v, 0)
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
	resultVector := &Vector{vec: result}
	if meta := v.Meta(); meta != nil {
		resultVector.attrs.Store(newVectorAttrs(meta))
	}
	return resultVector
}

func (v *Vector) Meta() IPersistentMap {
	if attrs := v.attrs.Load(); attrs != nil {
		return attrs.meta
	}
	return nil
}

func (v *Vector) WithMeta(meta IPersistentMap) any {
	attrs := v.attrs.Load()
	if attrs == nil && meta == nil || attrs != nil && attrs.meta == meta {
		return v
	}
	result := &Vector{vec: v.vec}
	if attrs == nil {
		result.attrs.Store(newVectorAttrs(meta))
	} else {
		result.attrs.Store(&vectorAttrs{meta: meta, flat: attrs.flat,
			hash: atomic.LoadUint32(&attrs.hash), hasheq: atomic.LoadUint32(&attrs.hasheq)})
	}
	return result
}

func (v *Vector) HashEq() uint32 {
	attrs := v.ensureAttrs()
	if hash := atomic.LoadUint32(&attrs.hasheq); hash != 0 {
		return hash
	}
	var hash uint32
	apersistentVectorHashEq(&hash, v)
	atomic.StoreUint32(&attrs.hasheq, hash)
	return hash
}

func (v *Vector) Hash() uint32 {
	attrs := v.ensureAttrs()
	if hash := atomic.LoadUint32(&attrs.hash); hash != 0 {
		return hash
	}
	var hash uint32
	apersistentVectorHash(&hash, v)
	atomic.StoreUint32(&attrs.hash, hash)
	return hash
}

func (v *Vector) ReduceInit(f IFn, init any) any {
	res := init
	for i := 0; i < v.Count(); i++ {
		res = Apply2(f, res, v.Nth(i))
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (v *Vector) Reduce(f IFn) any {
	if v.Count() == 0 {
		return Apply0(f)
	}
	res := v.ValAt(0)
	for i := 1; i < v.Count(); i++ {
		res = Apply2(f, res, v.Nth(i))
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (v *Vector) KVReduce(f IFn, init any) any {
	for i := 0; i < v.Count(); i++ {
		init = Apply3(f, init, i, v.Nth(i))
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
	if attrs := v.attrs.Load(); attrs != nil {
		return attrs
	}
	attrs := &vectorAttrs{}
	if v.attrs.CompareAndSwap(nil, attrs) {
		return attrs
	}
	return v.attrs.Load()
}

func (v *Vector) AsTransient() ITransientCollection {
	return &TransientVector{
		vec: vector.NewTransient(&v.vec),
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

func newVectorWithMeta(storage vector.Persistent, meta IPersistentMap) *Vector {
	result := &Vector{vec: storage}
	if meta != nil {
		result.attrs.Store(newVectorAttrs(meta))
	}
	return result
}
