package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/chzyer/readline"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/runtime"
	"github.com/glojurelang/glojure/value"
)

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
		o.env = runtime.NewEnvironment(runtime.WithStdout(o.stdout))
	}
	{ // switch to the namespace
		_, err := o.env.Eval(value.NewList([]interface{}{
			value.NewSymbol("ns"),
			value.NewSymbol(o.namespace),
		}))
		if err != nil {
			panic(err)
		}
	}

	defaultPrompt := func() string {
		curNS := "?"
		ns, err := o.env.Eval(value.NewSymbol("*ns*"))
		if err == nil {
			curNS = ns.(*value.Namespace).Name().String()
		}
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

		rdr := reader.New(strings.NewReader(expr), reader.WithFilename("repl"))
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

				val, err := o.env.Eval(val)
				if err != nil {
					return "", err
				}
				return value.ToString(val), nil
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
