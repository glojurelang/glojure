package runtime

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

func NewEnvironment(opts ...EvalOption) value.Environment {
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
		// TODO: reader shorthand for this (and pointers, maps, and channels)
		env.Define("go-sliceof", value.NewGoVal(func(t reflect.Type) *value.GoTyp {
			return value.NewGoTyp(reflect.SliceOf(t))
		}))

		////////////////////////////////////////////////////////////////////////////
		// basic types

		// numeric types
		{
			// integral types
			env.Define("int", value.NewGoTyp(reflect.TypeOf(int(0))))
			env.Define("uint", value.NewGoTyp(reflect.TypeOf(uint(0))))
			env.Define("uintptr", value.NewGoTyp(reflect.TypeOf(uintptr(0))))

			env.Define("int8", value.NewGoTyp(reflect.TypeOf(int8(0))))
			env.Define("int16", value.NewGoTyp(reflect.TypeOf(int16(0))))
			env.Define("int32", value.NewGoTyp(reflect.TypeOf(int32(0))))
			env.Define("int64", value.NewGoTyp(reflect.TypeOf(int64(0))))

			env.Define("uint8", value.NewGoTyp(reflect.TypeOf(uint8(0))))
			env.Define("uint16", value.NewGoTyp(reflect.TypeOf(uint16(0))))
			env.Define("uint32", value.NewGoTyp(reflect.TypeOf(uint32(0))))
			env.Define("uint64", value.NewGoTyp(reflect.TypeOf(uint64(0))))

			// floating point types
			env.Define("float32", value.NewGoTyp(reflect.TypeOf(float32(0))))
			env.Define("float64", value.NewGoTyp(reflect.TypeOf(float64(0))))

			// aliases
			env.Define("byte", value.NewGoTyp(reflect.TypeOf(byte(0))))
			env.Define("rune", value.NewGoTyp(reflect.TypeOf(rune(0))))
		}

		// string
		{
			env.Define("string", value.NewGoTyp(reflect.TypeOf("")))
		}

		// boolean
		{
			env.Define("bool", value.NewGoTyp(reflect.TypeOf(true)))
		}

		env.Define("error", value.NewGoTyp(reflect.TypeOf((*error)(nil)).Elem()))
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

	return env
}

func (p *Program) Eval(opts ...EvalOption) (value.Value, error) {
	env := NewEnvironment(opts...)

	for _, node := range p.nodes {
		_, err := env.Eval(node)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
