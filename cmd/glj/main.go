//go:build !wasm

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
	gljruntime "github.com/glojurelang/glojure/pkg/runtime"
)

const depsGenerator = "github.com/glojurelang/glojure/cmd/gljdeps"

func depsGeneratorPackage(version string) string {
	if version == "" || version == "0.0.0" || strings.Contains(version, "+dirty") {
		return depsGenerator
	}
	return depsGenerator + "@v" + version
}

func main() {
	raiseOpenFileLimit()

	if _, err := os.Stat("./gljdeps.edn"); err == nil {
		exe, err := exec.LookPath("go")
		if err != nil {
			panic(fmt.Errorf("failed to find `go` executable: %v", err))
		}

		cmd := exec.Command(exe, "run", depsGeneratorPackage(gljruntime.Version))
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("failed to generate Glojure dependencies: %v", err)
		}

		argv := append([]string{"go", "run", "./glj/cmd/glj"}, os.Args[1:]...)
		if err := syscall.Exec(exe, argv, os.Environ()); err != nil {
			log.Fatalf("failed to run %v: %v", exe, err)
		}
		panic("a successful exec syscall should replace this process")
	} else if !os.IsNotExist(err) {
		log.Fatal(err)
	}

	gljmain.Main(os.Args[1:])
}
