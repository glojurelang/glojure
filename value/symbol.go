package value

import "strings"

var (
	SymbolUnquote       = NewSymbol("clojure.core/unquote")
	SymbolSpliceUnquote = NewSymbol("splice-unquote")
)

type Symbol struct {
	Section
	Value string
}

// NewSymbol creates a new symbol.
func NewSymbol(s string, opts ...Option) *Symbol {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Symbol{
		Section: o.section,
		Value:   s,
	}
}

func (s *Symbol) Namespace() string {
	// Return the namespace of the symbol, or the empty string if it
	// doesn't have one.
	if i := strings.Index(s.Value, "/"); i != -1 {
		return s.Value[:i]
	}
	return ""
}

func (s *Symbol) Name() string {
	// Return the name of the symbol, or the empty string if it doesn't
	// have one.
	if i := strings.Index(s.Value, "/"); i != -1 {
		return s.Value[i+1:]
	}
	return s.Value
}

func (s *Symbol) FullName() string {
	return s.Value
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
	return s.Value
}

func (s *Symbol) Equal(v interface{}) bool {
	other, ok := v.(*Symbol)
	if !ok {
		return false
	}
	return s.Value == other.Value
}
