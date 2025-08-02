package lang

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
	kw   *intern.Value
	hash uint32
}

var (
	_ Hasher = Keyword{}
)

func NewKeyword(s string) Keyword {
	return Keyword{
		kw:   intern.GetByString(s),
		hash: Hash(s) ^ keywordHashMask,
	}
}

func InternKeywordSymbol(s *Symbol) Keyword {
	return NewKeyword(s.FullName())
}

func InternKeywordString(s string) Keyword {
	return NewKeyword(s)
}

func InternKeyword(ns, name interface{}) Keyword {
	return InternKeywordSymbol(InternSymbol(ns, name))
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

func (k Keyword) Equals(v interface{}) bool {
	return k == v
}

func (k Keyword) Invoke(args ...interface{}) interface{} {
	if len(args) == 0 || len(args) > 2 {
		panic(fmt.Errorf("wrong number of args (%v) passed to: %v", len(args), k))
	}
	var defaultVal interface{} = nil
	if len(args) == 2 {
		defaultVal = args[1]
	}

	assoc, ok := args[0].(Associative)
	if !ok {
		return defaultVal
	}

	entry := assoc.EntryAt(k)
	if entry == nil {
		return defaultVal
	}

	return entry.Val()
}

func (k Keyword) ApplyTo(args ISeq) interface{} {
	return k.Invoke(seqToSlice(args)...)
}

func (k Keyword) Hash() uint32 {
	return k.hash
}

func (k Keyword) Compare(other any) int {
	if otherKw, ok := other.(Keyword); ok {
		return strings.Compare(k.String(), otherKw.String())
	}
	panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare Keyword with %T", other)))
}
