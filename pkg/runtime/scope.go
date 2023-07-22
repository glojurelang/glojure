package runtime

import (
	value "github.com/glojurelang/glojure/pkg/lang"
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
