package main

import (
	"flag"

	"github.com/glojurelang/glojure/repl"
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		repl.Start()
	}
}
