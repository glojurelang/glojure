package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/glojurelang/glojure/internal/deps"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
)

func main() {
	dps, err := deps.Load()
	if err != nil {
		log.Fatal(err)
	}
	if dps != nil {
		if err := dps.Get(); err != nil {
			log.Fatalf("failed to fetch dependencies: %v", err)
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
