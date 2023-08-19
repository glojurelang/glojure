package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/repl"
	"github.com/glojurelang/glojure/pkg/runtime"

	"github.com/glojurelang/glojure/internal/deps"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
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

	args := os.Args[1:]

	runtime.AddLoadPath(os.DirFS("."))

	if len(args) == 0 {
		repl.Start()
	} else {
		file, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		env := lang.GlobalEnv

		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).BindRoot(lang.Seq(os.Args[2:]))

		rdr := reader.New(bufio.NewReader(file), reader.WithGetCurrentNS(func() *lang.Namespace {
			return env.CurrentNamespace()
		}))
		for {
			val, err := rdr.ReadOne()
			if err == reader.ErrEOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			_, err = env.Eval(val)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
