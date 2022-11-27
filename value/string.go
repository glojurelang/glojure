package value

import (
	"strconv"
)

// Str is a string.
type Str struct {
	Section
	Value string
}

func NewStr(s string, opts ...Option) *Str {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Str{
		Section: o.section,
		Value:   s,
	}
}

func (s *Str) String() string {
	// NB: java does not support \x escape sequences, but go does.  this
	// results in a difference in the output of the string from Clojure
	// if such characters make it into the string. We will escape them
	// but Clojure on the JVM will not.
	return strconv.Quote(s.Value)
}

func (s *Str) Count() int {
	return len(s.Value)
}

func (s *Str) Equal(v Value) bool {
	other, ok := v.(*Str)
	if !ok {
		return false
	}
	return s.Value == other.Value
}

func (s *Str) GoValue() interface{} {
	return s.Value
}
