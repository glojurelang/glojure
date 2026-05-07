//go:build windows

package repl

import (
	"os"
	"os/signal"
)

// termState is a no-op on Windows (no termios).
type termState struct{}

// saveTermState is a no-op on Windows.
func saveTermState() termState {
	return termState{}
}

// suspendProcess is a no-op on Windows (no job control signals).
func suspendProcess(_ termState, _ func() string) {}

// notifyInterrupt registers for os.Interrupt and returns the channel
// and a cleanup function.
func notifyInterrupt() (<-chan os.Signal, func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	return sigCh, func() { signal.Stop(sigCh) }
}
