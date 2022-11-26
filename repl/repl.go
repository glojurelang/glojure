package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/runtime"
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

	prompt := o.namespace + "=> "

	rl, err := readline.NewEx(&readline.Config{
		Prompt: prompt,
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
		}
		expr = ""
		rl.SetPrompt(prompt)

		for _, val := range vals {
			val, err := o.env.Eval(val)
			if err != nil {
				fmt.Fprintln(o.stdout, err)
				continue
			}
			fmt.Fprintln(o.stdout, val)
		}
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
