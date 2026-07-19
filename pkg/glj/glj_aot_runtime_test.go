//go:build glj_aot_runtime

package glj

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/runtime"
)

func TestCompactRuntimeNamespaceLoaders(t *testing.T) {
	for _, resource := range []string{"clojure/core", "glojure/go/io"} {
		if runtime.GetNSLoader(resource) == nil {
			t.Errorf("required namespace loader %q is not linked", resource)
		}
	}

	for _, resource := range []string{
		"clojure/core/async",
		"clojure/core/protocols",
		"clojure/string",
	} {
		if runtime.GetNSLoader(resource) != nil {
			t.Errorf("optional namespace loader %q is linked", resource)
		}
	}
}
