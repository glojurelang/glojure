package thread

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestCurrentThreadCompatibility(t *testing.T) {
	method, ok := pkgmap.Get("Thread.currentThread")
	if !ok {
		t.Fatal("Thread.currentThread is not registered")
	}
	current := lang.Apply0(method.(lang.IFn)).(*Thread)
	if got := len(current.GetStackTrace()); got != 0 {
		t.Fatalf("stack trace length = %d, want 0", got)
	}
}
