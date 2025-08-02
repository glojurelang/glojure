package lang

import (
	"fmt"
	"strings"
)

type Symbol struct {
	meta IPersistentMap
	ns   string
	name string
}

// NewSymbol creates a new symbol.
func NewSymbol(s string) *Symbol {
	ns, name := "", s

	idx := strings.Index(s, "/")
	if idx != -1 && s != "/" && s[0] != '/' {
		ns = s[:idx]
		name = s[idx+1:]
	}
	if !isValidSymbol(ns, name) {
		panic(NewIllegalArgumentError("invalid symbol: " + s))
	}
	return &Symbol{
		ns:   ns,
		name: name,
	}
}

func InternSymbol(ns, name interface{}) *Symbol {
	if ns == nil {
		return NewSymbol(name.(string))
	}
	if ns, ok := ns.(string); ok {
		if ns == "" {
			return NewSymbol(name.(string))
		}
	}
	return NewSymbol(ns.(string) + "/" + name.(string))
}

func (s *Symbol) Namespace() string {
	return s.ns
}

func (s *Symbol) Name() string {
	return s.name
}

func (s *Symbol) Compare(other any) int {
	otherSym, ok := other.(*Symbol)
	if !ok {
		panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare Symbol with %T", other)))
	}
	
	// Compare namespace first
	if s.ns != otherSym.ns {
		if s.ns == "" && otherSym.ns != "" {
			return -1
		}
		if s.ns != "" && otherSym.ns == "" {
			return 1
		}
		if nsComp := strings.Compare(s.ns, otherSym.ns); nsComp != 0 {
			return nsComp
		}
	}
	
	// Then compare name
	return strings.Compare(s.name, otherSym.name)
}

func (s *Symbol) FullName() string {
	return s.String()
}

func isValidSymbol(ns, name string) bool {
	var full string
	if ns == "" {
		full = name
	} else {
		full = ns + "/" + name
	}

	// early special case for the division operator /
	if full == "/" {
		return true
	}

	if name == "" {
		// empty name
		return false
	}
	if ns == "" && full[0] == '/' {
		// empty namespace
		return false
	}
	if strings.HasSuffix(name, ":") {
		// name ends with a colon (match clojure)
		return false
	}
	if strings.Contains(name, "::") {
		// name contains double colon
		//
		// NB: clojure reader rejects this, but clojure.core/symbol
		// accepts it
		return false
	}

	return true
}

func (s *Symbol) String() string {
	if s.ns == "" {
		return s.name
	}
	return s.ns + "/" + s.name
}

func (s *Symbol) Equals(v interface{}) bool {
	if s == v {
		return true
	}
	other, ok := v.(*Symbol)
	if !ok {
		return false
	}
	if other == nil {
		return false
	}
	return s.ns == other.ns && s.name == other.name
}

func (s *Symbol) Meta() IPersistentMap {
	return s.meta
}

func (s *Symbol) WithMeta(meta IPersistentMap) interface{} {
	if s.meta == meta {
		return s
	}

	symCopy := *s
	symCopy.meta = meta
	return &symCopy
}

func (s *Symbol) Hash() uint32 {
	h := getHash()
	h.Write([]byte(s.ns + "/" + s.name))
	return h.Sum32() ^ symbolHashMask
}
