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
	"github.com/glojurelang/glojure/value/numbers"

	"github.com/glojurelang/glojure/gen/gljimports"
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

	// bootstrap namespace control
	{
		// bootstrap implementation of the ns macro
		env.DefineMacro("ns", value.ApplyerFunc(func(env value.Environment, args []interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("ns: expected namespace name")
			}

			sym, ok := args[0].(*value.Symbol)
			if !ok {
				return nil, fmt.Errorf("ns: expected symbol as namespace name")
			}
			ns := env.FindOrCreateNamespace(sym)
			env.SetCurrentNamespace(ns)
			return ns, nil
		}))
		env.Define(value.NewSymbol("in-ns"), value.ApplyerFunc(func(env value.Environment, args []interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("in-ns: expected namespace name")
			}

			sym, ok := args[0].(*value.Symbol)
			if !ok {
				return nil, fmt.Errorf("in-ns: expected symbol as namespace name")
			}
			ns := env.FindOrCreateNamespace(sym)
			env.SetCurrentNamespace(ns)
			return ns, nil
		}))
	}

	gljimports.RegisterImports(func(name string, val interface{}) {
		env.Define(value.NewSymbol(name), val)
	})

	define := func(name string, val interface{}) {
		env.Define(value.NewSymbol(name), val)
	}

	{
		// go-sliceof returns a slice type with the given element type.
		// TODO: reader shorthand for this (and pointers, maps, and channels)
		define("go-sliceof", func(t reflect.Type) reflect.Type {
			return reflect.SliceOf(t)
		})

		////////////////////////////////////////////////////////////////////////////
		// basic types

		// numeric types
		{
			// integral types
			define("int", reflect.TypeOf(int(0)))
			define("uint", reflect.TypeOf(uint(0)))
			define("uintptr", reflect.TypeOf(uintptr(0)))

			define("int8", reflect.TypeOf(int8(0)))
			define("int16", reflect.TypeOf(int16(0)))
			define("int32", reflect.TypeOf(int32(0)))
			define("int64", reflect.TypeOf(int64(0)))

			define("uint8", reflect.TypeOf(uint8(0)))
			define("uint16", reflect.TypeOf(uint16(0)))
			define("uint32", reflect.TypeOf(uint32(0)))
			define("uint64", reflect.TypeOf(uint64(0)))

			// floating point types
			define("float32", reflect.TypeOf(float32(0)))
			define("float64", reflect.TypeOf(float64(0)))

			// aliases
			define("byte", reflect.TypeOf(byte(0)))
			define("rune", reflect.TypeOf(rune(0)))
		}
		// numeric functions
		{
			define("glojure.lang/AsNumber", value.AsNumber)

			define("glojure.lang.numbers/Inc", value.Inc)
			define("glojure.lang.numbers/IncP", value.IncP)
			define("glojure.lang.Numbers/Add", numbers.Add)
		}
		// iteration functions
		{
			define("glojure.lang.iteration/NewIterator", value.NewIterator)
			define("glojure.lang.iteration/NewRangeIterator", value.NewRangeIterator)

			define("glojure.lang.functional/Reduce", value.Reduce)
			define("glojure.lang.functional/ReduceInit", value.ReduceInit)

			define("glojure.lang.iteration/NewConcatIterator", value.NewConcatIterator)
		}
		{
			define("glojure.lang/FindNamespace", value.FindNamespace)
		}

		// string
		{
			define("string", reflect.TypeOf(""))
		}

		// boolean
		{
			define("bool", reflect.TypeOf(true))
		}

		define("error", reflect.TypeOf((*error)(nil)).Elem())
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
