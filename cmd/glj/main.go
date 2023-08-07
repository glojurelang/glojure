package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/glojurelang/glojure/pkg/lang"
	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/repl"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
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
		env := lang.GlobalEnv
		rdr := reader.New(bufio.NewReader(file), reader.WithGetCurrentNS(func() *value.Namespace {
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
