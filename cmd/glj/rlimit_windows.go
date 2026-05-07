//go:build windows

package main

// raiseOpenFileLimit is a no-op on Windows (no rlimit syscall).
func raiseOpenFileLimit() {}
