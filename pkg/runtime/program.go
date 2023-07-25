package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
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

	define := func(name string, val interface{}) {
		// TODO: use DefVar!
		env.BindLocal(value.NewSymbol(name), val)
	}

	{
		define("glojure.lang.Apply", value.IFnFunc(func(args ...interface{}) interface{} {
			if len(args) != 2 {
				panic(fmt.Errorf("wrong number of arguments (%d) to glojure.lang.Apply", len(args)))
			}
			res, err := value.Apply(args[0], seqToSlice(value.Seq(args[1])))
			if err != nil {
				panic(err)
			}
			return res
		}))

		define("glojure.lang.Import", func(args ...interface{}) {
			if len(args) != 1 {
				panic(fmt.Errorf("wrong number of arguments (%d) to glojure.lang.Import", len(args)))
			}

			export := args[0].(string)
			v, ok := pkgmap.Get(export)
			if !ok {
				// TODO: panic
				fmt.Println("WARNING: export not found in package map:", args[0], "- this will be a panic in the future")
				return
			}
			env.CurrentNamespace().Import(export, v)
		})

		// define("glojure.lang.Volatile", reflect.TypeOf(&value.Volatile{}))
		// define("glojure.lang.MultiFn", reflect.TypeOf(&value.MultiFn{}))
		// define("glojure.lang.Var", reflect.TypeOf(&value.Var{}))
		// define("glojure.lang.Atom", reflect.TypeOf(&value.Atom{}))

		// define("glojure.lang.Namespace", reflect.TypeOf(&value.Namespace{}))

		// define("glojure.lang.LockingTransaction", value.LockingTransaction)

		// define("glojure.lang.Hash", value.Hash)

		// define("big.Int", reflect.TypeOf(big.Int{}))
		// define("glojure.lang.BigDecimal", reflect.TypeOf(&value.BigDecimal{}))

		// TODO: go versions of these java-isms
		{
			{
				var x interface{}
				define("Throwable", reflect.TypeOf(x))
			}
			define("java.util.regex.Matcher", reflect.TypeOf(&regexp.Regexp{})) // wrong
			define("java.io.PrintWriter", reflect.TypeOf(&bytes.Buffer{}))      // wrong
		}

		// define("glojure.lang.AsInt64", value.AsInt64)
		// define("glojure.lang.AsFloat64", value.AsFloat64)
		define("glojure.lang.AsNumber", func(v interface{}) interface{} {
			x, ok := value.AsNumber(v)
			if !ok {
				panic(fmt.Errorf("cannot convert %T to number", v))
			}
			return x
		})
	}
	{
		define("glojure.lang.AppendWriter", func(w io.Writer, v interface{}) io.Writer {
			var err error
			switch v := v.(type) {
			case string:
				_, err = w.Write([]byte(v))
			case []byte:
				_, err = w.Write(v)
			case rune:
				_, err = w.Write([]byte{byte(v)})
			case value.Char:
				_, err = w.Write([]byte{byte(v)})
			default:
				err = fmt.Errorf("unsupported type %T", v)
			}

			if err != nil {
				panic(err)
			}
			return w
		})
		define("glojure.lang.WriteWriter", func(w io.Writer, v interface{}) io.Writer {
			var err error
			switch v := v.(type) {
			case string:
				_, err = w.Write([]byte(v))
			case []byte:
				_, err = w.Write(v)
			default:
				err = fmt.Errorf("unsupported type %T", v)
			}
			if err != nil {
				panic(err)
			}
			return w
		})
		// define("glojure.lang.CharAt", func(s string, idx int) value.Char {
		// 	return value.NewChar([]rune(s)[idx])
		// })
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
