//go:build wasm

package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"
)

const debugMode = false
const cpuProfile = false

// Start starts the REPL.
func Start(opts ...Option) {
	o := options{
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		namespace: "user",
	}
	for _, opt := range opts {
		opt(&o)
	}

	if o.env == nil {
		o.env = initEnv(o.stdout)
	}
	{ // set namespace
		_, err := o.env.Eval(lang.NewList(
			lang.NewSymbol("ns"),
			lang.NewSymbol(o.namespace),
		))
		if err != nil {
			panic(err)
		}
	}

	defaultPrompt := func() string {
		curNS := "?"
		ns := o.env.CurrentNamespace()
		curNS = ns.Name().String()
		return curNS + "=> "
	}

	rl := bufio.NewReader(os.Stdin)

	var expr string

	fmt.Print(defaultPrompt())
	for {
		line, err := rl.ReadString('\n')
		if err != nil {
			break
		}
		expr += line + "\n"

		rdr := reader.New(strings.NewReader(expr), reader.WithFilename("repl"), reader.WithGetCurrentNS(func() *lang.Namespace {
			return o.env.CurrentNamespace()
		}))

		vals, err := rdr.ReadAll()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Print("... ")
				continue
			}
			fmt.Fprintln(o.stdout, err)
		}
		expr = ""
		for _, val := range vals {
			out, err := func() (out string, err error) {
				defer func() {
					if panicErr := recover(); panicErr != nil {
						err = fmt.Errorf("panic: %v\nstacktrace:\n%s", panicErr, string(debug.Stack()))
					}
				}()

				//runtime.Debug = true
				val, err := o.env.Eval(val)
				runtime.Debug = false
				if err != nil {
					return "", err
				}
				return lang.PrintString(val), nil
			}()
			if err != nil {
				fmt.Fprintln(o.stdout, err)
				continue
			}
			fmt.Fprintln(o.stdout, out)
		}
		fmt.Print(defaultPrompt())
	}
}

func readLine(r io.Reader) (string, error) {
	var line string
	for {
		buf := make([]byte, 1)
		if _, err := r.Read(buf); err != nil {
			return "", err
		}
		if buf[0] == '\n' {
			break
		}
		line += string(buf)
	}
	return line, nil
}

func initEnv(stdout io.Writer) lang.Environment {
	if cpuProfile {
		f, err := os.Create("./gljInitEnvCpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	startTime := time.Now()

	// TODO: clean up this code. copied from rtcompat.go.
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*lang.Var{lang.VarCurrentNS, lang.VarWarnOnReflection, lang.VarUncheckedMath, lang.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	lang.PushThreadBindings(lang.NewMap(kvs...))

	env := runtime.NewEnvironment(runtime.WithStdout(stdout))
	if debugMode {
		fmt.Printf("Environment created in %v\n", time.Since(startTime))
	}

	return env
}
