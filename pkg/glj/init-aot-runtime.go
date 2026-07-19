//go:build !glj_no_aot_stdlib && glj_aot_runtime

package glj

import (
	// Compact AOT executables always need clojure.core. Applications opt into
	// other precompiled namespace loaders with blank imports, allowing Go's
	// linker to omit standard-library namespaces the program never uses.
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core"
	// clojure.core/slurp and spit lazily resolve this namespace.
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io"
)
