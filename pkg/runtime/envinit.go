package runtime

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/glojurelang/glojure/pkg/lang"
)

type evalOptions struct {
	stdout   io.Writer
	stderr   io.Writer
	loadPath []string
	env      *environment
}

type EvalOption func(*evalOptions)

func WithStdout(w io.Writer) EvalOption {
	return func(opts *evalOptions) {
		opts.stdout = w
	}
}

func WithStderr(w io.Writer) EvalOption {
	return func(opts *evalOptions) {
		opts.stderr = w
	}
}

func WithLoadPath(path []string) EvalOption {
	return func(opts *evalOptions) {
		opts.loadPath = path
	}
}

func withEnv(env lang.Environment) EvalOption {
	e := env.(*environment)
	return func(opts *evalOptions) {
		opts.env = e
	}
}

func NewEnvironment(opts ...EvalOption) lang.Environment {
	options := &evalOptions{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	for _, opt := range opts {
		opt(options)
	}

	env := options.env
	if env == nil {
		env = newEnvironment(context.Background(), options.stdout, options.stderr)
		env.loadPath = options.loadPath
	}
	// this is rather rather hacky
	lang.GlobalEnv = env

	// bootstrap namespace control
	{
		// bootstrap implementation of the ns macro
		env.DefVar(lang.NewSymbol("in-ns"), lang.NewFnFunc(func(args ...interface{}) interface{} {
			if len(args) != 1 {
				panic(fmt.Errorf("in-ns: expected namespace name"))
			}

			sym, ok := args[0].(*lang.Symbol)
			if !ok {
				panic(fmt.Errorf("in-ns: expected symbol as namespace name"))
			}
			ns := lang.FindOrCreateNamespace(sym)
			env.SetCurrentNamespace(ns)
			return ns
		}))
	}

	// Add stdlib
	RT.Load("glojure/core")

	return env
}
