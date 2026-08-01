//go:build unix

package repl

import (
	"os"

	"golang.org/x/sys/unix"
)

// IsTerminal reports whether standard input accepts terminal ioctls.
func IsTerminal() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), ioctlGetTermios)
	return err == nil
}
