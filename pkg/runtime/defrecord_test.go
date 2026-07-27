//go:build !glj_aot_runtime

package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestDefrecordCreatesRecordConstructors(t *testing.T) {
	env := NewEnvironment().(*environment)
	previousNS := env.CurrentNamespace()
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, previousNS))
	t.Cleanup(lang.PopThreadBindings)

	result := ReadEval(`
		(ns runtime.defrecord-test)
		(defrecord Point [x y])
		(let [point (->Point 1 2)
		      moved (assoc point :x 3)
		      tagged (assoc moved :tag :origin)]
		  [(record? point)
		   (:x moved)
		   (:tag tagged)
		   (record? tagged)
		   (record? (dissoc tagged :y))])
	`)
	expected := lang.NewVector(true, int64(3), lang.NewKeyword("origin"), true, false)
	if !lang.Equals(result, expected) {
		t.Fatalf("defrecord result = %v, want %v", result, expected)
	}

	ns := lang.FindNamespace(lang.NewSymbol("runtime.defrecord-test"))
	constructor := ns.FindInternedVar(lang.NewSymbol("->Point")).Deref()
	recordConstructor, ok := constructor.(*lang.RecordConstructor)
	if !ok {
		t.Fatalf("constructor type = %T, want *lang.RecordConstructor", constructor)
	}
	if got := recordConstructor.RecordType().FieldNames(); len(got) != 2 ||
		got[0] != "x" || got[1] != "y" {
		t.Fatalf("record fields = %v, want [x y]", got)
	}
}
