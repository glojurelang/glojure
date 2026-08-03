package boolean

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestBooleanHostConstructor(t *testing.T) {
	class := lang.NewClass(reflectBool, "java.lang.Boolean")
	if got := lang.NewHostInstance(class, "true"); got != true {
		t.Fatalf("new Boolean(true) = %v, want true", got)
	}
	if got := lang.NewHostInstance(class, "yes"); got != false {
		t.Fatalf("new Boolean(yes) = %v, want false", got)
	}
}
