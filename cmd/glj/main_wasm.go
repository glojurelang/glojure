//go:build wasm

package main

import (
	"os"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
)

func main() {
	gljmain.Main(os.Args[1:])
}
