package lang

import (
	"fmt"
	"strings"
	"sync"

	"go4.org/intern"
)

// Keyword represents a keyword. Syntactically, a keyword is a symbol
// that starts with a colon and evaluates to itself.
type Keyword struct {
	// kw is an interned string. This guarantees that two keywords with
	// the same name share the underlying string.
	kw   *intern.Value
	hash uint32
}

var (
	_ Hasher = Keyword{}

	keywordRegistry   = make(map[string]struct{})
	keywordRegistryMu sync.RWMutex
)

// keywordLookup lets persistent maps accept a Keyword without first boxing it
// into an interface value. Keyword invocation is a common map access path, and
// boxing the value-form Keyword otherwise allocates once per lookup.
type keywordLookup interface {
	valAtKeyword(Keyword, any) any
}

func NewKeyword(s string) Keyword {
	keywordRegistryMu.Lock()
	keywordRegistry[s] = struct{}{}
	keywordRegistryMu.Unlock()

	return Keyword{
		kw:   intern.GetByString(s),
		hash: Hash(s) ^ keywordHashMask,
	}
}

// AllKeywords returns all keyword strings that have been interned.
func AllKeywords() []string {
	keywordRegistryMu.RLock()
	defer keywordRegistryMu.RUnlock()
	result := make([]string, 0, len(keywordRegistry))
	for k := range keywordRegistry {
		result = append(result, k)
	}
	return result
}

func InternKeywordSymbol(s *Symbol) Keyword {
	return NewKeyword(s.FullName())
}

func InternKeywordString(s string) Keyword {
	return NewKeyword(s)
}

func InternKeyword(ns, name interface{}) Keyword {
	return InternKeywordSymbol(InternSymbol(ns, name))
}

func (k Keyword) value() string {
	return k.kw.Get().(string)
}

func (k Keyword) Namespace() any {
	// Return the namespace of the keyword, or nil if it doesn't have
	// one.
	// TODO: support both nil and empty string namespace as clojure does
	if i := strings.Index(k.value(), "/"); i != -1 {
		return k.value()[:i]
	}
	return nil
}

func (k Keyword) Name() string {
	// Return the name of the keyword, or the empty string if it
	// doesn't have one.
	if i := strings.Index(k.value(), "/"); i != -1 {
		return k.value()[i+1:]
	}
	return k.value()
}

func (k Keyword) Sym() *Symbol {
	return InternSymbol(k.Namespace(), k.Name())
}

func (k Keyword) String() string {
	return ":" + k.value()
}

func (k Keyword) Equals(v interface{}) bool {
	return k == v
}

// EqualsKeyword compares a dynamically represented value with an unboxed
// keyword. Generated dispatch paths use it to avoid boxing the known keyword
// solely for an equality check.
func EqualsKeyword(value any, keyword Keyword) bool {
	got, ok := value.(Keyword)
	return ok && got == keyword
}

func (k Keyword) Invoke(args ...interface{}) interface{} {
	if len(args) == 0 || len(args) > 2 {
		panic(fmt.Errorf("wrong number of args (%v) passed to: %v", len(args), k))
	}
	if len(args) == 2 {
		return k.Invoke2(args[0], args[1])
	}
	return k.Invoke1(args[0])
}

func (k Keyword) Invoke1(coll interface{}) interface{} {
	return k.Invoke2(coll, nil)
}

func (k Keyword) Invoke2(coll, defaultVal interface{}) interface{} {
	if lookup, ok := coll.(keywordLookup); ok {
		return lookup.valAtKeyword(k, defaultVal)
	}
	lookup, ok := coll.(ILookup)
	if !ok {
		return defaultVal
	}
	return lookup.ValAtDefault(k, defaultVal)
}

func (k Keyword) ApplyTo(args ISeq) interface{} {
	return k.Invoke(seqToSlice(args)...)
}

func (k Keyword) Hash() uint32 {
	return k.hash
}

func (k Keyword) Compare(other any) int {
	if otherKw, ok := other.(Keyword); ok {
		s := k.String()
		os := otherKw.String()
		if s == os {
			return 0
		}
		ns, ok := k.Namespace().(string)
		if !ok {
			if otherKw.Namespace() != nil {
				return -1
			}
		} else {
			ons, ok := otherKw.Namespace().(string)
			if !ok {
				return 1
			}
			nsc := strings.Compare(ns, ons)
			if nsc != 0 {
				return nsc
			}
		}
		return strings.Compare(k.Name(), otherKw.Name())
	}
	panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare Keyword with %T", other)))
}
