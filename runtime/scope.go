package runtime

import "github.com/glojurelang/glojure/value"

type scope struct {
	parent *scope
	syms   map[string]value.Value
}

func newScope() *scope {
	return &scope{syms: make(map[string]value.Value)}
}

func (s *scope) define(name string, val value.Value) {
	s.syms[name] = val
}

func (s *scope) push() *scope {
	return &scope{parent: s, syms: make(map[string]value.Value)}
}

func (s *scope) lookup(name string) (value.Value, bool) {
	if v, ok := s.syms[name]; ok {
		return v, true
	}
	if s.parent == nil {
		return nil, false
	}
	return s.parent.lookup(name)
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
