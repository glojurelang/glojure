package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/repl"
	"github.com/glojurelang/glojure/runtime"
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		repl.Start()
	} else if flag.NArg() == 1 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		rdr := reader.New(bufio.NewReader(file))
		vals, err := rdr.ReadAll()
		if err != nil {
			log.Fatal(err)
		}
		env := runtime.NewEnvironment(runtime.WithStdout(os.Stdout))
		for _, val := range vals {
			_, err := env.Eval(val)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
