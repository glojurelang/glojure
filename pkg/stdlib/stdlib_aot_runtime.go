//go:build glj_aot_runtime

// Package stdlib provides the standard library for the mratlang language.
package stdlib

import "embed"

// StdLib is empty in compact AOT executables. Every required namespace must
// have a linked AOT loader, so embedding fallback source would be redundant.
var StdLib embed.FS
