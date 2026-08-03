package instant

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestFullyQualifiedInstantParse(t *testing.T) {
	parse, found := pkgmap.Get("java.time.Instant.parse")
	if !found {
		t.Fatal("java.time.Instant.parse was not registered")
	}
	if got := lang.Apply1(parse, "2006-03-24T15:49:00Z"); got == nil {
		t.Fatal("Instant.parse returned nil")
	}
}
