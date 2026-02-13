//go:build wasm

package main

import (
	"os"

	// Bootstrap the runtime
	_ "github.com/gloathub/glojure/pkg/glj"
	"github.com/gloathub/glojure/pkg/gljmain"
)

func main() {
	gljmain.Main(os.Args[1:])
}
