//go:build !glj_no_aot_stdlib && !glj_aot_runtime

package glj

import (
	// Add NS loaders for the standard library.
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core/async"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core/protocols"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/string"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io"
)
