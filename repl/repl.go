package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/runtime"
	value "github.com/glojurelang/glojure/pkg/lang"

	// pprof
	"net/http"
	_ "net/http/pprof"
)

const debugMode = true
const cpuProfile = false

func init() {
	// start pprof
	if debugMode {
		go func() {
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				fmt.Println("pprof start failed:", err)
			}
		}()
		// shell command to examine pprof profile with a web ui:
		// $ go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile
	}
}

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
		_, err := o.env.Eval(value.NewList(
			value.NewSymbol("ns"),
			value.NewSymbol(o.namespace),
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

	rl, err := readline.NewEx(&readline.Config{
		Prompt: defaultPrompt(),
		//DisableAutoSaveHistory: true,
		Stdin:  io.NopCloser(o.stdin),
		Stdout: o.stdout,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	var expr string

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		expr += line + "\n"

		rdr := reader.New(strings.NewReader(expr), reader.WithFilename("repl"), reader.WithGetCurrentNS(func() *value.Namespace {
			return o.env.CurrentNamespace()
		}))

		vals, err := rdr.ReadAll()
		if err != nil {
			if errors.Is(err, io.EOF) {
				rl.SetPrompt("... ")
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
				return value.PrintString(val), nil
			}()
			if err != nil {
				fmt.Fprintln(o.stdout, err)
				continue
			}
			fmt.Fprintln(o.stdout, out)
		}
		rl.SetPrompt(defaultPrompt())
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

func initEnv(stdout io.Writer) value.Environment {
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
	for _, vr := range []*value.Var{value.VarCurrentNS, value.VarWarnOnReflection, value.VarUncheckedMath, value.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	value.PushThreadBindings(value.NewMap(kvs...))

	env := runtime.NewEnvironment(runtime.WithStdout(stdout))
	if debugMode {
		fmt.Printf("Environment created in %v\n", time.Since(startTime))
	}

	return env
}
