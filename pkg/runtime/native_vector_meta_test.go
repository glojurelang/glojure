package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

func TestNativeVecDropsMetadataAndPreservesValues(t *testing.T) {
	meta := lang.NewMap("tag", true)
	source := lang.NewVector(1, 2).WithMeta(meta).(*lang.Vector)
	result := nativeVec(source).(lang.IPersistentVector)
	if !lang.Equals(result, source) || result.(lang.IMeta).Meta() != nil || source.Meta() != meta {
		t.Fatal("vec changed values or mutated input metadata")
	}
	if nativeVec(result) != result {
		t.Fatal("vec copied a plain persistent vector")
	}
	if !lang.Equals(nativeVec(lang.NewMapEntry("key", 1)), lang.NewVector("key", 1)) {
		t.Fatal("vec changed a map entry")
	}
}

func TestNativeIntoSmallVectorsRetainsMetadataAndInputSnapshots(t *testing.T) {
	namespace := lang.FindNamespace(lang.NewSymbol("clojure.core"))
	fn := namespace.FindInternedVar(lang.NewSymbol("into")).Get()
	meta := lang.NewMap("tag", true)
	target := lang.NewVector(1).WithMeta(meta).(*lang.Vector)
	source := lang.NewVector(2, 3, 4)
	result := lang.Apply2(fn, target, source).(lang.IPersistentVector)
	if !lang.Equals(result, lang.NewVector(1, 2, 3, 4)) || result.(lang.IMeta).Meta() != meta ||
		target.Count() != 1 || source.Count() != 3 {
		t.Fatal("into changed metadata or an input snapshot")
	}
}
