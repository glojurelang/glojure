//go:build !glj_no_goimports

package glj

// Add the Go standard library to the pkgmap (needed for REPL, not AOT).
import _ "github.com/glojurelang/glojure/pkg/gen/gljimports"
