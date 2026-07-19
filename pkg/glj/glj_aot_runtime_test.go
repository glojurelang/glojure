//go:build glj_aot_runtime

package glj

import (
	"fmt"
	"strings"
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

func TestCompactRuntimeDoesNotLinkHTTPURLSupport(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "no URL opener is linked") {
			t.Fatalf("slurp panic = %v, want missing URL opener error", recovered)
		}
	}()

	Var("clojure.core", "slurp").Invoke("https://example.com/data")
}
