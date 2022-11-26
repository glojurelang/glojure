// Package stdlib provides the standard library for the mratlang language.
package stdlib

import (
	"embed"
)

//go:embed mratfiles
var StdLib embed.FS
