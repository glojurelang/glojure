package uuid

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	guuid "github.com/google/uuid"
)

func TestIsUUID(t *testing.T) {
	if !IsUUID(RandomUUID()) {
		t.Fatal("RandomUUID result is not recognized as a UUID")
	}
	if !IsUUID(guuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")) {
		t.Fatal("google/uuid parser result is not recognized as a UUID")
	}
	if IsUUID("not-a-uuid") {
		t.Fatal("string was recognized as a UUID")
	}
}

func TestUUIDClassAcceptsCompatibilityAndParserValues(t *testing.T) {
	class, ok := pkgmap.HostClass("UUID")
	if !ok {
		t.Fatal("UUID host class is not registered")
	}
	if !lang.HasType(class, RandomUUID()) {
		t.Fatal("UUID class rejected a compatibility UUID")
	}
	if !lang.HasType(class, guuid.New()) {
		t.Fatal("UUID class rejected a parser UUID")
	}
}
