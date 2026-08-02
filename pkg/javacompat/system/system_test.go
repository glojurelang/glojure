package system

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestGetPropertiesContainsSeededProperties(t *testing.T) {
	properties, ok := GetProperties().(lang.IPersistentMap)
	if !ok {
		t.Fatalf("GetProperties returned %T, want persistent map", GetProperties())
	}
	if got := properties.ValAt("file.separator"); got == nil {
		t.Fatal("file.separator is missing")
	}
}
