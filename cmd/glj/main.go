package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"log"
	"os"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/repl"
	"github.com/glojurelang/glojure/runtime"
	"github.com/glojurelang/glojure/value"
	"github.com/jtolio/gls"
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
		gls.EnsureGoroutineId(func(uint) {
			env := initEnv(os.Stdout)
			rdr := reader.New(bufio.NewReader(file), reader.WithGetCurrentNS(func() string {
				return env.CurrentNamespace().Name().String()
			}))
			for {
				val, err := rdr.ReadOne()
				if errors.Is(err, io.EOF) {
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
		})
	}
}

func initEnv(stdout io.Writer) value.Environment {
	env := runtime.NewEnvironment(runtime.WithStdout(stdout))
	// TODO: clean up this code. copied from rtcompat.go.
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*value.Var{value.VarCurrentNS, value.VarWarnOnReflection, value.VarUncheckedMath} {
		kvs = append(kvs, vr, vr.Deref())
	}
	value.PushThreadBindings(value.NewMap(kvs...))

	return env
}
