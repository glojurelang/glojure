package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/stdlib"
)

type Program struct {
	nodes []interface{}
}

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

func withEnv(env value.Environment) EvalOption {
	e := env.(*environment)
	return func(opts *evalOptions) {
		opts.env = e
	}
}

func NewEnvironment(opts ...EvalOption) value.Environment {
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
	// TODO: this is rather rather hacky
	value.GlobalEnv = env

	// bootstrap namespace control
	{
		// bootstrap implementation of the ns macro
		env.DefVar(value.NewSymbol("in-ns"), value.IFnFunc(func(args ...interface{}) interface{} {
			if len(args) != 1 {
				panic(fmt.Errorf("in-ns: expected namespace name"))
			}

			sym, ok := args[0].(*value.Symbol)
			if !ok {
				panic(fmt.Errorf("in-ns: expected symbol as namespace name"))
			}
			ns := value.FindOrCreateNamespace(sym)
			env.SetCurrentNamespace(ns)
			return ns
		}))
	}

	{
		// Add stdlib
		evalFile := func(path string) {
			core, err := stdlib.StdLib.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("could not read stdlib core.glj: %v", err))
			}
			r := reader.New(strings.NewReader(string(core)), reader.WithFilename(path), reader.WithGetCurrentNS(func() *value.Namespace {
				return env.CurrentNamespace()
			}))

			for {
				expr, err := r.ReadOne()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					panic(fmt.Sprintf("error reading core lib %v: %v", path, err))
				}
				_, err = env.Eval(expr)
				if err != nil {
					panic(fmt.Sprintf("error evaluating core lib %v: %v", path, err))
				}
			}
		}
		evalFile("glojure/core.glj")
	}

	return env
}

func (p *Program) Eval(opts ...EvalOption) (interface{}, error) {
	env := NewEnvironment(opts...)

	for _, node := range p.nodes {
		_, err := env.Eval(node)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
