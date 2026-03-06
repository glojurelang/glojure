//go:build !wasm

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/gloathub/glojure/internal/deps"

	// Bootstrap the runtime
	_ "github.com/gloathub/glojure/pkg/glj"
	"github.com/gloathub/glojure/pkg/gljmain"
)

func main() {
	// Raise the open-file limit; dep type-checking opens many files.
	var rl syscall.Rlimit
	if syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl) == nil && rl.Cur < 4096 {
		rl.Cur = 4096
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rl) //nolint:errcheck
	}

	dps, err := deps.Load()
	if err != nil {
		log.Fatal(err)
	}
	if dps != nil {
		if err := dps.Gen(); err != nil {
			panic(err)
		}

		exe, err := exec.LookPath("go")
		if err != nil {
			panic(fmt.Errorf("failed to find `go` executable: %v", err))
		}

		argv := append([]string{"go", "run", "./glj/cmd/glj"}, os.Args[1:]...)
		if err := syscall.Exec(exe, argv, os.Environ()); err != nil {
			log.Fatalf("failed to run %v: %v", exe, err)
		}
		panic("a successful exec syscall should replace this process")
	}

	gljmain.Main(os.Args[1:])
}
