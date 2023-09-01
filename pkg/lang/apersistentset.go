package lang

import "github.com/glojurelang/glojure/internal/murmur3"

type (
	APersistentSet interface {
		AFn
		IPersistentSet
		IHashEq
	}
)

func apersistentsetEquiv(a APersistentSet, o any) bool {
	set, ok := o.(IPersistentSet) // TODO: more general?
	if !ok {
		return false
	}

	if a.Count() != set.Count() {
		return false
	}

	for s := a.Seq(); s != nil; s = s.Next() {
		if !set.Contains(s.First()) {
			return false
		}
	}
	return true
}

func apersistentsetHashEq(hc *uint32, a APersistentSet) uint32 {
	if *hc != 0 {
		return *hc
	}

	hash := murmur3.HashUnordered(seqToInternalSeq(a.Seq()), HashEq)
	*hc = hash
	return hash
}
