package value

type (
	// Counter is an interface for compound values whose elements can be
	// counted.
	Counter interface {
		Count() int
	}

	// Conjer is an interface for values that can be conjed onto.
	Conjer interface {
		Conj(...interface{}) Conjer
	}

	Associative interface {
		ContainsKey(interface{}) bool
		EntryAt(interface{}) (interface{}, bool)
	}

	IPersistentStack interface {
		Peek() interface{}
		Pop() IPersistentStack
	}

	Equaler interface {
		Equal(interface{}) bool
	}

	IPersistentMap interface {
		Equaler // Note: not in Clojure's interfaces

		//Iterable do we need this?
		Associative
		Counter

		Assoc(k, v interface{}) IPersistentMap

		// AssocEx is like Assoc, but returns an error if the key already
		// exists.
		AssocEx(key, val interface{}) (IPersistentMap, error)

		// Without returns a new map with the given key removed.
		Without(key interface{}) IPersistentMap
	}

	Comparer interface {
		Compare(other interface{}) int
	}
)
