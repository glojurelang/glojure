//go:build !wasm && !plan9

package main

import "syscall"

// raiseOpenFileLimit bumps the soft open-file limit to at least 4096;
// dep type-checking opens many files concurrently.
func raiseOpenFileLimit() {
	var rl syscall.Rlimit
	if syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl) == nil && rl.Cur < 4096 {
		rl.Cur = 4096
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rl) //nolint:errcheck
	}
}
