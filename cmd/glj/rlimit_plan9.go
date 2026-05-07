//go:build plan9

package main

// raiseOpenFileLimit is a no-op on Plan 9 (no rlimit syscall).
func raiseOpenFileLimit() {}
