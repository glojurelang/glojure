package value

import (
	"fmt"
	"strings"

	"go4.org/intern"
)

// Keyword represents a keyword. Syntactically, a keyword is a symbol
// that starts with a colon and evaluates to itself.
type Keyword struct {
	// kw is an interned string. This guarantees that two keywords with
	// the same name share the underlying string.
	kw *intern.Value
}

func NewKeyword(s string) Keyword {
	return Keyword{
		kw: intern.GetByString(s),
	}
}

func (k Keyword) value() string {
	return k.kw.Get().(string)
}

func (k Keyword) Namespace() string {
	// Return the namespace of the keyword, or the empty string if it
	// doesn't have one.
	if i := strings.Index(k.value(), "/"); i != -1 {
		return k.value()[:i]
	}
	return ""
}

func (k Keyword) Name() string {
	// Return the name of the keyword, or the empty string if it
	// doesn't have one.
	if i := strings.Index(k.value(), "/"); i != -1 {
		return k.value()[i+1:]
	}
	return k.value()
}

func (k Keyword) String() string {
	return ":" + k.value()
}

func (k Keyword) Equal(v interface{}) bool {
	return k == v
}

func (k Keyword) Apply(env Environment, args []interface{}) (interface{}, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("wrong number of args (%v) passed to: %v", len(args), k)
	}
	var defaultVal interface{} = nil
	if len(args) == 2 {
		defaultVal = args[1]
	}

	assoc, ok := args[0].(Associative)
	if !ok {
		return defaultVal, nil
	}
	v, ok := assoc.EntryAt(k)
	if !ok {
		return defaultVal, nil
	}
	return v, nil
}
