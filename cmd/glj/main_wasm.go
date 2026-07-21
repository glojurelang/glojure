//go:build wasm

package main

import (
	"os"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
	_ "github.com/glojurelang/glojure/pkg/gljmain/interactive"
)

func main() {
	gljmain.Main(os.Args[1:])
}
