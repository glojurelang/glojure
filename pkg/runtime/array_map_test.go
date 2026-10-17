package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

func TestNativeArrayMapOrderDuplicatesAndApply(t *testing.T) {
	fn := nativeArrayMap{}
	pairs := []any{"a", 1, "b", nil, "a", 3}
	direct := fn.Invoke(pairs...).(lang.IPersistentMap)
	applied := fn.ApplyTo(lang.NewList(pairs...)).(lang.IPersistentMap)
	if !lang.Equals(direct, applied) || direct.Count() != 2 || direct.ValAt("a") != 3 ||
		direct.Seq().First().(lang.IMapEntry).Key() != "a" {
		t.Fatal("array-map changed duplicate handling or key order")
	}
	for _, m := range []any{fn.Invoke0(), fn.ApplyTo(nil)} {
		if m.(lang.IPersistentMap).Count() != 0 {
			t.Fatal("empty array-map was not empty")
		}
	}
}

func TestNativeArrayMapRejectsTrailingEntry(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("array-map accepted odd argument count")
		}
	}()
	nativeArrayMap{}.ApplyTo(lang.NewList("a", 1, lang.NewVector("b", 2)))
}
