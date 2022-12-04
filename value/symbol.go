package value

import "strings"

var (
	SymbolUnquote       = NewSymbol("clojure.core/unquote")
	SymbolSpliceUnquote = NewSymbol("splice-unquote")
)

type Symbol struct {
	Section
	ns   string
	name string
}

// NewSymbol creates a new symbol.
func NewSymbol(s string, opts ...Option) *Symbol {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	ns, name := "", s

	idx := strings.Index(s, "/")
	if idx != -1 && s != "/" && s[0] != '/' {
		ns = s[:idx]
		name = s[idx+1:]
	}
	return &Symbol{
		Section: o.section,
		ns:      ns,
		name:    name,
	}
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
	return s.ns == other.ns && s.name == other.name
}
