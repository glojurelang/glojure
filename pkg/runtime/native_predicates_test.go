package runtime

import (
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeCorePredicatesMatchHostTypes(t *testing.T) {
	type namedString string
	for _, predicate := range []struct {
		name   string
		target reflect.Type
	}{
		{"string?", reflect.TypeOf("")},
		{"char?", reflect.TypeOf(lang.Char(0))},
		{"map?", reflect.TypeOf((*lang.IPersistentMap)(nil)).Elem()},
		{"vector?", reflect.TypeOf((*lang.IPersistentVector)(nil)).Elem()},
		{"set?", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem()},
		{"seq?", reflect.TypeOf((*lang.ISeq)(nil)).Elem()},
	} {
		variable := lang.NSCore.FindInternedVar(lang.NewSymbol(predicate.name))
		if variable == nil {
			t.Skip("core not loaded in this runtime mode")
		}
		fn := variable.Get()
		for _, value := range []any{
			nil, false, int64(1), "value", namedString("value"), lang.Char('x'),
			lang.NewMap(), lang.NewVector(), lang.NewSet(), lang.NewList(1),
			map[string]int{}, []any{}, (*lang.Vector)(nil), (*lang.LazySeq)(nil),
		} {
			want := lang.HasType(predicate.target, value)
			if got := lang.Apply1(fn, value); got != want {
				t.Errorf("%s(%T) = %v, want %v", predicate.name, value, got, want)
			}
		}
	}
}
