package value

import (
	"fmt"
	"reflect"
)

type (
	Object interface {
		Hash() uint32
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
		Invoke(args ...interface{}) interface{}
		ApplyTo(args ISeq) interface{}
	}

	IReduce interface {
		Reduce(f IFn) interface{}
	}

	IReduceInit interface {
		ReduceInit(f IFn, init interface{}) interface{}
	}

	// IMeta is an interface for values that can have metadata.
	IMeta interface {
		// Meta returns the metadata associated with this value.
		Meta() IPersistentMap
	}

	IObj interface {
		IMeta

		// WithMeta returns a new value with the given metadata.
		WithMeta(meta IPersistentMap) interface{}
	}

	// Counted is an interface for compound values whose elements can be
	// counted.
	Counted interface {
		Count() int
	}

	// Conjer is an interface for values that can be conjed onto.
	Conjer interface {
		Conj(interface{}) Conjer
	}

	ISeqable interface {
		Seq() ISeq
	}

	IMapEntry interface {
		Key() interface{}
		Val() interface{}
	}

	Associative interface {
		IPersistentCollection
		ILookup

		ContainsKey(interface{}) bool

		EntryAt(interface{}) IMapEntry

		Assoc(k, v interface{}) Associative
	}

	ILookup interface {
		ValAt(interface{}) interface{}
		ValAtDefault(interface{}, interface{}) interface{}
	}

	Equaler interface {
		Equal(interface{}) bool
	}

	Reversible interface {
		RSeq() ISeq
	}

	IPending interface {
		IsRealized() bool
	}

	Indexed interface {
		Nth(int) (interface{}, bool)
		NthDefault(int, interface{}) interface{}
	}

	//////////////////////////////////////////////////////////////////////////////
	// Collections

	ITransientCollection interface {
		Conj(interface{}) Conjer
		Persistent() IPersistentCollection
	}

	IEditableCollection interface {
		AsTransient() ITransientCollection
	}

	IPersistentCollection interface {
		ISeqable
		Counted
		// NB: we diverge from Clojure here, which has a cons method,
		// which is used by the conj runtime method. I expect the cons
		// methods are a relic of a previous implementation.
		Conjer

		IsEmpty() bool

		// Equiv(interface{}) bool
	}

	IPersistentStack interface {
		Peek() interface{}
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
		Equaler // Note: not in Clojure's interfaces

		//Iterable do we need this?
		Associative
		Counted

		// AssocEx is like Assoc, but returns an error if the key already
		// exists.
		AssocEx(key, val interface{}) IPersistentMap

		// Without returns a new map with the given key removed.
		Without(key interface{}) IPersistentMap
	}

	// IPersistentVector is a persistent vector.
	IPersistentVector interface {
		Sequential

		Equaler // Note: not in Clojure's interfaces

		Associative
		IPersistentStack
		Reversible
		Indexed
		Counted // Note: not in Clojure's vector interface, oddly

		Length() int

		AssocN(int, interface{}) IPersistentVector

		Cons(interface{}) IPersistentVector
	}

	IPersistentSet interface {
		IPersistentCollection
		Counted

		Disjoin(interface{}) IPersistentSet
		Contains(interface{}) bool
		Get(interface{}) interface{}
	}

	ITransientSet interface {
		IPersistentCollection
		Counted

		Disjoin(interface{}) ITransientSet
		Contains(interface{}) bool
		Get(interface{}) interface{}
	}

	ISeq interface {
		Sequential

		ISeqable

		// First returns the first element of the sequence.
		First() interface{}

		// Next returns the rest of the sequence, or nil if there are no
		// more.
		Next() ISeq

		// More returns true if there are more elements in the sequence.
		More() ISeq

		// TODO: Missing: Cons, IPersistentCollection
	}

	IChunk interface {
		Indexed

		DropFirst() IChunk
		Reduce(fn IFn, init interface{}) interface{}
	}

	IChunkedSeq interface {
		ISeq

		ChunkedFirst() IChunk
		ChunkedNext() ISeq
		ChunkedMore() ISeq
	}

	Comparer interface {
		Compare(other interface{}) int
	}

	// References

	IDeref interface {
		Deref() interface{}
	}

	IRef interface {
		IDeref

		SetValidator(vf IFn)
		Validator() IFn
		Watches() IPersistentMap
		AddWatch(key interface{}, fn IFn)
		RemoveWatch(key interface{})
	}

	IAtom interface {
		Swap(f IFn, args ISeq) interface{}
		CompareAndSet(oldv, newv interface{}) bool
		Reset(newVal interface{}) interface{}
	}

	IAtom2 interface {
		IAtom
		// IPersistentVector swapVals(IFn f);
		// IPersistentVector swapVals(IFn f, Object arg);
		// IPersistentVector swapVals(IFn f, Object arg1, Object arg2);
		// IPersistentVector swapVals(IFn f, Object x, Object y, ISeq args);
		// IPersistentVector resetVals(Object newv);

	}
)

func Conj(coll Conjer, x interface{}) Conjer {
	if coll == nil {
		return emptyList.Conj(x)
	}
	return coll.Conj(x)
}

// WithMeta returns a new value with the given metadata.
func WithMeta(v interface{}, meta IPersistentMap) (interface{}, error) {
	// TODO: just take an IObj
	iobj, ok := v.(IObj)
	if !ok {
		return nil, fmt.Errorf("value of type %T can't have metadata", v)
	}
	return iobj.WithMeta(meta), nil
}

func Assoc(a interface{}, k, v interface{}) Associative {
	if a == nil {
		return NewMap(k, v)
	}
	assoc, ok := a.(Associative)
	if !ok {
		panic(fmt.Errorf("value of type %T can't be assoc'd", a))
	}
	return assoc.Assoc(k, v)
}

func Dissoc(x interface{}, k interface{}) interface{} {
	if x == nil {
		return nil
	}
	return x.(IPersistentMap).Without(k)
}

func Get(coll, key interface{}) interface{} {
	return GetDefault(coll, key, nil)
}

func GetDefault(coll, key, def interface{}) interface{} {
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
	if reflect.TypeOf(coll).Kind() == reflect.Slice {
		if idx, ok := AsInt(key); ok {
			res, ok := Nth(coll, idx)
			if ok {
				return res
			}
		}
	}
	return def
}

func Count(coll interface{}) int {
	switch arg := coll.(type) {
	case nil:
		return 0
	case string:
		return len(arg)
	case Counted:
		return arg.Count()
	}
	seq, ok := Seq(coll).(ISeq)
	if !ok {
		panic(fmt.Errorf("count expects a collection, got %v", coll))
	}
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
