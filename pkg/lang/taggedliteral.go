package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// TaggedLiteral is the portable representation returned for an EDN reader tag.
type TaggedLiteral struct {
	tag  *Symbol
	form any
	meta IPersistentMap
}

var _ ILookup = (*TaggedLiteral)(nil)
var _ IObj = (*TaggedLiteral)(nil)

func NewTaggedLiteral(tag, form any) any {
	return &TaggedLiteral{tag: MustHostCast[*Symbol](tag), form: form}
}

func (t *TaggedLiteral) Tag() *Symbol { return t.tag }
func (t *TaggedLiteral) Form() any    { return t.form }

func (t *TaggedLiteral) ValAt(key any) any {
	return t.ValAtDefault(key, nil)
}

func (t *TaggedLiteral) ValAtDefault(key, notFound any) any {
	if keyword, ok := key.(Keyword); ok {
		switch keyword.Name() {
		case "tag":
			return t.tag
		case "form":
			return t.form
		}
	}
	return notFound
}

func (t *TaggedLiteral) Meta() IPersistentMap { return t.meta }

func (t *TaggedLiteral) WithMeta(meta IPersistentMap) any {
	if t.meta == meta {
		return t
	}
	copy := *t
	copy.meta = meta
	return &copy
}

func (t *TaggedLiteral) Equals(other any) bool {
	value, ok := other.(*TaggedLiteral)
	return ok && Equals(t.tag, value.tag) && Equals(t.form, value.form)
}

func (t *TaggedLiteral) Equiv(other any) bool { return t.Equals(other) }

func (t *TaggedLiteral) String() string {
	return fmt.Sprintf("#%s %s", t.tag, PrintString(t.form))
}

func init() {
	class := NewClass(reflect.TypeOf((*TaggedLiteral)(nil)),
		"clojure.lang.TaggedLiteral")
	pkgmap.SetHostClassPackage("TaggedLiteral", "clojure.lang")
	pkgmap.SetHostClass("TaggedLiteral", class)
	pkgmap.Set("clojure.lang.TaggedLiteral.create", FnFunc2(NewTaggedLiteral))
}
