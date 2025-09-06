//go:build !glj_no_aot_stdlib

package glj

import (
	// Add NS loaders for the standard library.
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/core"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/core/async"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/protocols"
)
