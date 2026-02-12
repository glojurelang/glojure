//go:build !glj_no_aot_stdlib

package glj

import (
	// Add NS loaders for the standard library.
	_ "github.com/ingydotnet/glojure/pkg/stdlib/clojure/core"
	_ "github.com/ingydotnet/glojure/pkg/stdlib/clojure/core/async"
	_ "github.com/ingydotnet/glojure/pkg/stdlib/clojure/core/protocols"
	_ "github.com/ingydotnet/glojure/pkg/stdlib/glojure/go/io"
)
