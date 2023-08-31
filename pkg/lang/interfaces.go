package lang

import (
	"fmt"
	"reflect"
	"time"
)

type (
	Object any

	// Hasher is an interface for types that can be hashed. It's not in
	// Clojure, but it's useful for Go where values don't come with a
	// default hash method.
	Hasher interface {
		Hash() uint32
	}

	// TODO: use this interface
	IHashEq interface {
		HashEq() uint32
	}

	Sequential interface {
		// Private interface method used to tag sequential types.
		xxx_sequential() // TODO: anything that inherits from ASeq in java impl should implement this
	}

	Named interface {
		Name() string
		Namespace() string
	}

	IRecord interface {
		xxx_irecord()
	}

	IDrop interface {
		Drop(n int) Sequential
	}

	IFn interface {
		Invoke(args ...any) any
		ApplyTo(args ISeq) any
	}

	IReduce interface {
		Reduce(f IFn) any
	}

	IReduceInit interface {
		ReduceInit(f IFn, init any) any
	}

	// IMeta is an interface for values that can have metadata.
	IMeta interface {
		// Meta returns the metadata associated with this value.
		Meta() IPersistentMap
	}

	IObj interface {
		IMeta

		// WithMeta returns a new value with the given metadata.
		WithMeta(meta IPersistentMap) any
	}

	// Counted is an interface for compound values whose elements can be
	// counted.
	Counted interface {
		Count() int
	}

	// Conser is an interface for values that can be consed onto.
	Conser interface {
		Cons(any) Conser
	}

	Seqable interface {
		Seq() ISeq
	}

	IMapEntry interface {
		Key() any
		Val() any
	}

	Associative interface {
		IPersistentCollection
		ILookup

		ContainsKey(any) bool

		EntryAt(any) IMapEntry

		Assoc(k, v any) Associative
	}

	ILookup interface {
		ValAt(any) any
		ValAtDefault(any, any) any
	}

	// Not a Clojure interface, but useful for Go
	Equalser interface {
		Equals(any) bool
	}
	// Not a Clojure interface, but useful for Go
	Equiver interface {
		Equiv(any) bool
	}

	Reversible interface {
		RSeq() ISeq
	}

	IPending interface {
		IsRealized() bool
	}

	Indexed interface {
		Counted

		Nth(int) any
		NthDefault(int, any) any
	}

	//////////////////////////////////////////////////////////////////////////////
	// Collections

	ITransientCollection interface {
		Conj(any) ITransientCollection
		Persistent() IPersistentCollection
	}

	IEditableCollection interface {
		AsTransient() ITransientCollection
	}

	IPersistentCollection interface {
		Seqable
		Counted
		Conser

		Empty() IPersistentCollection

		Equiv(any) bool
	}

	IPersistentStack interface {
		Peek() any
		Pop() IPersistentStack
	}

	IPersistentList interface {
		Sequential

		IPersistentCollection
		IPersistentStack
		// Clojure's IPersistentList does not implement this, but it
		// likely should.
		ISeq
	}

	IPersistentMap interface {
		//Iterable do we need this?
		Associative
		Counted

		// AssocEx is like Assoc, but returns an error if the key already
		// exists.
		AssocEx(key, val any) IPersistentMap

		// Without returns a new map with the given key removed.
		Without(key any) IPersistentMap
	}

	// IPersistentVector is a persistent vector.
	IPersistentVector interface {
		Sequential

		Associative
		IPersistentStack
		Reversible
		Indexed
		Counted // Note: not in Clojure's vector interface, oddly

		Length() int

		AssocN(int, any) IPersistentVector

		Cons(any) IPersistentVector
	}

	IPersistentSet interface {
		IPersistentCollection
		Counted

		Disjoin(any) IPersistentSet
		Contains(any) bool
		Get(any) any
	}

	ITransientSet interface {
		IPersistentCollection
		Counted

		Disjoin(any) ITransientSet
		Contains(any) bool
		Get(any) any
	}

	ISeq interface {
		IPersistentCollection
		Conser

		// First returns the first element of the sequence.
		First() any

		// Next returns the rest of the sequence, or nil if there are no
		// more.
		Next() ISeq

		// More returns true if there are more elements in the sequence.
		More() ISeq
	}

	IChunk interface {
		Indexed

		DropFirst() IChunk
		ReduceInit(fn IFn, init any) any
	}

	IChunkedSeq interface {
		ISeq
		Sequential

		ChunkedFirst() IChunk
		ChunkedNext() ISeq
		ChunkedMore() ISeq
	}

	Comparer interface {
		Compare(other any) int
	}

	// References

	IDeref interface {
		Deref() any
	}

	IBlockingDeref interface {
		DerefWithTimeout(timeoutMS int64, timeoutVal any) any
	}

	IRef interface {
		IDeref

		SetValidator(vf IFn)
		Validator() IFn
		Watches() IPersistentMap
		AddWatch(key any, fn IFn)
		RemoveWatch(key any)
	}

	IAtom interface {
		Swap(f IFn, args ISeq) any
		CompareAndSet(oldv, newv any) bool
		Reset(newVal any) any
	}

	IAtom2 interface {
		IAtom
		// IPersistentVector swapVals(IFn f);
		// IPersistentVector swapVals(IFn f, Object arg);
		// IPersistentVector swapVals(IFn f, Object arg1, Object arg2);
		// IPersistentVector swapVals(IFn f, Object x, Object y, ISeq args);
		// IPersistentVector resetVals(Object newv);

	}

	////////////////////////////////////////////////////////////////////////////
	// Abstract classes
	//
	// TODO: represent Clojure's abstract classes as interfaces to
	// provide compile-time checks for implementations of required
	// methods.

	// Java Future interface
	Future interface {
		Get() any
		GetWithTimeout(timeout int64, timeUnit time.Duration) any
		// Cancel(mayInterruptIfRunning bool) bool
		// IsCancelled() bool
		// IsDone() bool
	}
)

var (
	// sentinel value for "not found"
	notFound = &struct{}{}
)

func Conj(coll Conser, x any) Conser {
	if coll == nil {
		return emptyList.Cons(x)
	}
	return coll.Cons(x)
}

// WithMeta returns a new value with the given metadata.
func WithMeta(v any, meta IPersistentMap) (any, error) {
	// TODO: just take an IObj
	iobj, ok := v.(IObj)
	if !ok {
		return nil, fmt.Errorf("value of type %T can't have metadata", v)
	}
	return iobj.WithMeta(meta), nil
}

func Assoc(a any, k, v any) Associative {
	if a == nil {
		return NewMap(k, v)
	}
	assoc, ok := a.(Associative)
	if !ok {
		panic(fmt.Errorf("value of type %T can't be assoc'd", a))
	}
	return assoc.Assoc(k, v)
}

func Dissoc(x any, k any) any {
	if x == nil {
		return nil
	}
	return x.(IPersistentMap).Without(k)
}

func Get(coll, key any) any {
	return GetDefault(coll, key, nil)
}

func GetDefault(coll, key, def any) any {
	switch arg := coll.(type) {
	case nil:
		return def
	case ILookup:
		return arg.ValAtDefault(key, def)
	case Associative:
		if arg.ContainsKey(key) {
			return arg.EntryAt(key).Val()
		}
	case IPersistentSet:
		if arg.Contains(key) {
			return arg.Get(key)
		}
	case ITransientSet:
		if arg.Contains(key) {
			return arg.Get(key)
		}
	case string:
		if idx, ok := AsInt(key); ok {
			res, ok := Nth(arg, idx)
			if ok {
				return res
			}
		}
	}
	collVal := reflect.ValueOf(coll)
	if collVal.Kind() == reflect.Slice {
		if idx, ok := AsInt(key); ok {
			res, ok := Nth(coll, idx)
			if ok {
				return res
			}
		}
	}
	if collVal.Kind() == reflect.Map {
		keyVal := reflect.ValueOf(key)
		res := collVal.MapIndex(keyVal)
		if res.IsValid() {
			return res.Interface()
		}
	}
	return def
}

func Count(coll any) int {
	switch arg := coll.(type) {
	case nil:
		return 0
	case string:
		return len(arg)
	case Counted:
		return arg.Count()
	}
	seq := Seq(coll)
	count := 0
	for ; seq != nil; seq = seq.Next() {
		count++
	}
	return count
}

func Keys(m Associative) ISeq {
	return NewMapKeySeq(Seq(m))
}

func Vals(m Associative) ISeq {
	return NewMapValSeq(Seq(m))
}

func Subvec(v IPersistentVector, start, end int) IPersistentVector {
	// if(end < start || start < 0 || end > v.count())
	// 	throw new IndexOutOfBoundsException();
	// if(start == end)
	// 	return PersistentVector.EMPTY;
	// return new APersistentVector.SubVector(null, v, start, end);
	if end < start || start < 0 || end > v.Count() {
		panic(fmt.Errorf("index out of bounds"))
	}
	if start == end {
		return emptyVector
	}
	return NewSubVector(nil, v, start, end)
}
