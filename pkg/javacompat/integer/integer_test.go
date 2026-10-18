package integer

import (
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestGoIntegerZeroAllocation(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[int32]()} {
		t.Run(typ.String(), func(t *testing.T) {
			got := reflect.ValueOf(lang.NewHostInstance(typ))
			if got.Type() != reflect.PointerTo(typ) || !got.Elem().IsZero() {
				t.Fatalf("new %v = %v, want a pointer to its zero value", typ, got)
			}
		})
	}
}

func TestIntegerClassConstructor(t *testing.T) {
	for _, name := range []string{"Integer", "java.lang.Integer"} {
		t.Run(name, func(t *testing.T) {
			class, ok := pkgmap.Get(name)
			if !ok {
				t.Fatalf("class %s is not registered", name)
			}
			for _, arg := range []any{"42", int64(42)} {
				if got := lang.NewHostInstance(class, arg); got != int32(42) {
					t.Fatalf("new %s(%v) = %v, want int32(42)", name, arg, got)
				}
			}
			t.Run("zero-arity", func(t *testing.T) {
				defer func() {
					want := "Integer constructor: wrong number of args (0)"
					if got := recover(); got != want {
						t.Fatalf("panic = %v, want Integer arity error", got)
					}
				}()
				lang.NewHostInstance(class)
			})
		})
	}
}
