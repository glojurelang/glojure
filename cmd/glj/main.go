package main

import (
	"bufio"
	"flag"
	"io"
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
		env := runtime.NewEnvironment(runtime.WithStdout(os.Stdout))
		rdr := reader.New(bufio.NewReader(file), reader.WithGetCurrentNS(func() string {
			return env.CurrentNamespace().Name().String()
		}))
		for {
			val, err := rdr.ReadOne()
			if err == io.EOF {
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
