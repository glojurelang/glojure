//go:build !glj_aot_runtime

package glj

// Retain HTTP URL support by default. Compact AOT executables can opt into
// this adapter explicitly when they use slurp with HTTP URLs.
import _ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io/http"
