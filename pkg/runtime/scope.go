package runtime

import "github.com/glojurelang/glojure/pkg/lang"

const inlineScopeBindings = 4

type scopeBinding struct {
	name  string
	value interface{}
}

type scope struct {
	parent   *scope
	count    int
	bindings [inlineScopeBindings]scopeBinding
	overflow map[string]interface{}
}

func newScope() *scope {
	return &scope{}
}

func (s *scope) define(sym *lang.Symbol, val interface{}) {
	name := sym.String()
	for i := 0; i < s.count; i++ {
		if s.bindings[i].name == name {
			s.bindings[i].value = val
			return
		}
	}
	if s.count < len(s.bindings) {
		s.bindings[s.count] = scopeBinding{name: name, value: val}
		s.count++
		return
	}
	if s.overflow == nil {
		s.overflow = make(map[string]interface{})
	}
	s.overflow[name] = val
}

func (s *scope) push() *scope {
	return &scope{parent: s}
}

func (s *scope) lookup(sym *lang.Symbol) (interface{}, bool) {
	return s.lookupName(sym.String())
}

func (s *scope) lookupName(name string) (interface{}, bool) {
	for i := 0; i < s.count; i++ {
		if s.bindings[i].name == name {
			return s.bindings[i].value, true
		}
	}
	if s.overflow != nil {
		if v, ok := s.overflow[name]; ok {
			return v, true
		}
	}
	if s.parent == nil {
		return nil, false
	}
	return s.parent.lookupName(name)
}

func (s *scope) reset(parent *scope) {
	for i := 0; i < s.count; i++ {
		s.bindings[i] = scopeBinding{}
	}
	s.count = 0
	clear(s.overflow)
	s.parent = parent
}
