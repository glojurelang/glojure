package glj

import (
	"testing"

	"github.com/glojurelang/glojure/value"
)

func TestGLJ(t *testing.T) {
	mp := Var("glojure.core", "map")
	inc := Var("glojure.core", "inc")
	res := value.PrintString(mp.Invoke(inc, Read("[1 2 3]")))
	if res != "(2 3 4)" {
		t.Errorf("Expected (2 3 4), got %v", res)
	}
}
