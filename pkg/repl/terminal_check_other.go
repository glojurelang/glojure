//go:build !unix

package repl

import (
	"os"
	"strings"
)

// IsTerminal reports whether standard input is an interactive character
// device on platforms without Unix terminal ioctls.
func IsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	name := strings.ToLower(info.Name())
	return name != "null" && name != "nul"
}
