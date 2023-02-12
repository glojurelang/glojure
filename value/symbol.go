package value

import (
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

func (s *Symbol) FullName() string {
	return s.String()
}

func (s *Symbol) IsValidFormat() bool {
	// early special case for the division operator /
	if s.FullName() == "/" {
		return true
	}

	ns, name := s.Namespace(), s.Name()
	if name == "" {
		// empty name
		return false
	}
	if ns == "" && s.FullName()[0] == '/' {
		// empty namespace
		return false
	}
	if strings.HasSuffix(name, ":") {
		// name ends with a colon (match clojure)
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

func (s *Symbol) Equal(v interface{}) bool {
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
	if Equal(s.meta, meta) {
		return s
	}

	symCopy := *s
	symCopy.meta = meta
	return &symCopy
}
