package mapentry

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestCreateAndRegistration(t *testing.T) {
	entry := Create("key", "value")
	if entry.Key() != "key" || entry.Val() != "value" {
		t.Fatalf("Create returned [%v %v]", entry.Key(), entry.Val())
	}

	for _, name := range []string{"MapEntry.create", "clojure.lang.MapEntry.create"} {
		value, ok := pkgmap.Get(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		create, ok := value.(func(any, any) *lang.MapEntry)
		if !ok {
			t.Fatalf("%s has type %T", name, value)
		}
		if got := create("k", "v"); got.Key() != "k" || got.Val() != "v" {
			t.Fatalf("registered Create returned [%v %v]", got.Key(), got.Val())
		}
	}
}
