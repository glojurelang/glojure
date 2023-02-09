package runtime

import (
	"github.com/glojurelang/glojure/value"
)

type scope struct {
	parent *scope
	syms   map[string]interface{}
}

func newScope() *scope {
	return &scope{syms: make(map[string]interface{})}
}

func (s *scope) define(sym *value.Symbol, val interface{}) {
	s.syms[sym.String()] = val
}

func (s *scope) push() *scope {
	return &scope{parent: s, syms: make(map[string]interface{})}
}

func (s *scope) lookup(sym *value.Symbol) (interface{}, bool) {
	if v, ok := s.syms[sym.String()]; ok {
		return v, true
	}
	if s.parent == nil {
		return nil, false
	}
	return s.parent.lookup(sym)
}

func (s *scope) printIndented(indent string) string {
	str := ""
	for k, v := range s.syms {
		str += indent + k + ": " + value.ToString(v) + "\n"
	}
	if s.parent != nil {
		str += s.parent.printIndented(indent + "  ")
	}
	return str
}
