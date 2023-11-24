package main

import (
	"os"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
)

func main() {
	// dps, err := deps.Load()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if dps != nil {
	// 	if err := dps.Gen(); err != nil {
	// 		panic(err)
	// 	}

	// 	exe, err := exec.LookPath("go")
	// 	if err != nil {
	// 		panic(fmt.Errorf("failed to find `go` executable: %v", err))
	// 	}

	// 	argv := append([]string{"go", "run", "./glj/cmd/glj"}, os.Args[1:]...)
	// 	if err := syscall.Exec(exe, argv, os.Environ()); err != nil {
	// 		log.Fatalf("failed to run %v: %v", exe, err)
	// 	}
	// 	panic("a successful exec syscall should replace this process")
	// }

	gljmain.Main(os.Args[1:])
}
