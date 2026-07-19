//go:build !glj_aot_runtime

// Package stdlib provides the standard library for the mratlang language.
package stdlib

import (
	"embed"
)

// Embed only Glojure source. Embedding the directories themselves also includes
// generated loader Go files, which are already compiled into AOT-enabled
// executables and are not runtime load-path resources.
//
//go:embed */*.glj */*/*.glj */*/*/*.glj
var StdLib embed.FS
