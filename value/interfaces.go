package value

import (
	"fmt"
	"reflect"
)

type (
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

	// Counter is an interface for compound values whose elements can be
	// counted.
	Counter interface {
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
		Conj(interface{}) ITransientCollection
		Persistent() IPersistentCollection
	}

	IEditableCollection interface {
		AsTransient() ITransientCollection
	}

	IPersistentCollection interface {
		ISeqable
		Counter
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
		Counter

		// AssocEx is like Assoc, but returns an error if the key already
		// exists.
		AssocEx(key, val interface{}) (IPersistentMap, error)

		// Without returns a new map with the given key removed.
		Without(key interface{}) IPersistentMap
	}

	// IPersistentVector is a persistent vector.
	IPersistentVector interface {
		Equaler // Note: not in Clojure's interfaces

		Associative
		IPersistentStack
		Reversible
		Indexed
		Counter // Note: not in Clojure's vector interface, oddly

		Length() int

		AssocN(int, interface{}) IPersistentVector

		Cons(interface{}) IPersistentVector
	}

	IPersistentSet interface {
		IPersistentCollection
		Counter

		Disjoin(interface{}) IPersistentSet
		Contains(interface{}) bool
		Get(interface{}) interface{}
	}

	ITransientSet interface {
		IPersistentCollection
		Counter

		Disjoin(interface{}) ITransientSet
		Contains(interface{}) bool
		Get(interface{}) interface{}
	}

	ISeq interface {
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
		Reduce(fn Applyer, init interface{}) interface{}
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

		SetValidator(vf Applyer)
		Validator() Applyer
		Watches() IPersistentMap
		AddWatch(key interface{}, fn Applyer)
		RemoveWatch(key interface{})
	}

	IAtom interface {
		// Object swap(IFn f);
		// Object swap(IFn f, Object arg);
		// Object swap(IFn f, Object arg1, Object arg2);
		// Object swap(IFn f, Object x, Object y, ISeq args);
		// boolean compareAndSet(Object oldv, Object newv);
		// Object reset(Object newval);
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
		return arg.ValAt(key)
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
	case Counter:
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
