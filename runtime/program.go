package mratlang

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/stdlib"
	"github.com/glojurelang/glojure/value"

	"github.com/glojurelang/glojure/gen/gljimports"
)

type Program struct {
	nodes []value.Value
}

type evalOptions struct {
	stdout   io.Writer
	loadPath []string
	env      *environment
}

type EvalOption func(*evalOptions)

func WithStdout(w io.Writer) EvalOption {
	return func(opts *evalOptions) {
		opts.stdout = w
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

func (p *Program) Eval(opts ...EvalOption) (value.Value, error) {
	options := &evalOptions{
		stdout: os.Stdout,
	}
	for _, opt := range opts {
		opt(options)
	}

	env := options.env
	if env == nil {
		env = newEnvironment(context.Background(), options.stdout)
		env.loadPath = options.loadPath
	}

	gljimports.RegisterImports(func(name string, val value.Value) {
		env.Define(name, val)
	})
	{
		// go-sliceof returns a slice type with the given element type.
		env.Define("go-sliceof", value.NewGoVal(func(t reflect.Type) *value.GoTyp {
			return value.NewGoTyp(reflect.SliceOf(t))
		}))

		// basic types
		env.Define("go/byte", value.NewGoTyp(reflect.TypeOf(byte(0))))
		env.Define("go/string", value.NewGoTyp(reflect.TypeOf("")))
		env.Define("go/int", value.NewGoTyp(reflect.TypeOf(int(0))))
	}

	{
		// Add stdlib

		core, err := stdlib.StdLib.ReadFile("mratfiles/core.mrat")
		if err != nil {
			panic(fmt.Sprintf("could not read stdlib core.mrat: %v", err))
		}
		r := reader.New(strings.NewReader(string(core)))
		exprs, err := r.ReadAll()
		if err != nil {
			panic(fmt.Sprintf("error reading core lib: %v", err))
		}
		for _, expr := range exprs {
			_, err := env.Eval(expr)
			if err != nil {
				panic(fmt.Sprintf("error evaluating core lib: %v", err))
			}
		}
	}

	for _, node := range p.nodes {
		_, err := env.Eval(node)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
