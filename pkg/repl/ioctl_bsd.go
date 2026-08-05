//go:build dragonfly || freebsd || netbsd || openbsd

package repl

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TIOCGETA
	ioctlSetTermios = unix.TIOCSETA
)
