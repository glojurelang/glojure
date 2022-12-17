// Package stdlib provides the standard library for the mratlang language.
package stdlib

import (
	"embed"
)

//go:embed glojure
var StdLib embed.FS
