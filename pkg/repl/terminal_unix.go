//go:build !wasm && !plan9

package repl

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// termState holds saved terminal state for suspend/resume.
type termState struct {
	fd     int
	cooked *unix.Termios
}

// saveTermState captures the current terminal state for later restore.
func saveTermState() termState {
	fd := int(os.Stdin.Fd())
	cooked, _ := unix.IoctlGetTermios(fd, ioctlGetTermios)
	return termState{fd: fd, cooked: cooked}
}

// suspendProcess switches to cooked mode, sends SIGTSTP, and waits for
// SIGCONT before restoring raw mode and redisplaying the prompt.
func suspendProcess(ts termState, prompt func() string) {
	rawState, _ := unix.IoctlGetTermios(ts.fd, ioctlGetTermios)
	unix.IoctlSetTermios(ts.fd, ioctlSetTermios, ts.cooked)
	fmt.Print("\r\n")

	contCh := make(chan os.Signal, 1)
	signal.Notify(contCh, syscall.SIGCONT)
	signal.Reset(syscall.SIGTSTP)
	syscall.Kill(syscall.Getpid(), syscall.SIGTSTP)
	<-contCh
	signal.Stop(contCh)

	unix.IoctlSetTermios(ts.fd, ioctlSetTermios, rawState)
	fmt.Print(prompt())
}

// notifyInterrupt registers for SIGINT and returns the channel and a
// cleanup function.
func notifyInterrupt() (<-chan os.Signal, func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	return sigCh, func() { signal.Stop(sigCh) }
}
