//go:build !glj_no_aot_stdlib && glj_aot_runtime

package glj

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"

	// Compact AOT executables always need clojure.core. Applications opt into
	// other precompiled namespace loaders with blank imports, allowing Go's
	// linker to omit standard-library namespaces the program never uses.
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core"
	// clojure.core/slurp and spit lazily resolve this namespace.
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io"
)

func init() {
	// Runtime expansion of defprotocol needs only this small subset of the
	// general Go import registry, which compact AOT executables omit.
	pkgmap.Set(
		"github.com/glojurelang/glojure/pkg/lang.NewProtocolMultiFn",
		lang.NewProtocolMultiFn,
	)
	pkgmap.Set(
		"github.com/glojurelang/glojure/pkg/lang.*MultiFn",
		reflect.TypeOf((*lang.MultiFn)(nil)),
	)
}
