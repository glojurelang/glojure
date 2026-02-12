//go:build wasm

package main

import (
	"os"

	// Bootstrap the runtime
	_ "github.com/ingydotnet/glojure/pkg/glj"
	"github.com/ingydotnet/glojure/pkg/gljmain"
)

func main() {
	gljmain.Main(os.Args[1:])
}
