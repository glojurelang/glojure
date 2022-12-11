package value

import "fmt"

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

	Associative interface {
		ContainsKey(interface{}) bool
		EntryAt(interface{}) (interface{}, bool)
		Assoc(k, v interface{}) Associative
	}

	IPersistentStack interface {
		Peek() interface{}
		Pop() IPersistentStack
	}

	Equaler interface {
		Equal(interface{}) bool
	}

	Reversible interface {
		RSeq() ISeq
	}

	Indexed interface {
		Nth(int) (interface{}, bool)
		NthDefault(int, interface{}) interface{}
	}

	IPersistentCollection interface {
		Counter
		// NB: we diverge from Clojure here, which has a cons method,
		// which is used by the conj runtime method. I expect the cons
		// methods are a relic of a previous implementation.
		Conjer

		IsEmpty() bool

		// Equiv(interface{}) bool
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

	Comparer interface {
		Compare(other interface{}) int
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
	iobj, ok := v.(IObj)
	if !ok {
		return nil, fmt.Errorf("value of type %T can't have metadata", v)
	}
	return iobj.WithMeta(meta), nil
}
