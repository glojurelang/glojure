package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/stdlib"
	"github.com/glojurelang/glojure/value"
	"github.com/glojurelang/glojure/value/util"

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
			ns := env.FindOrCreateNamespace(sym)
			env.SetCurrentNamespace(ns)
			return ns
		}))
	}

	gljimports.RegisterImports(func(name string, val interface{}) {
		// TODO: use DefVar!
		env.BindLocal(value.NewSymbol(name), val)
	})

	define := func(name string, val interface{}) {
		// TODO: use DefVar!
		env.BindLocal(value.NewSymbol(name), val)
	}

	{
		// go-sliceof returns a slice type with the given element type.
		// TODO: reader shorthand for this (and pointers, maps, and channels)
		define("go-sliceof", func(t reflect.Type) reflect.Type {
			return reflect.SliceOf(t)
		})
		define("go-pointerto", func(t reflect.Type) reflect.Type {
			return reflect.PtrTo(t)
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

			// define("glojure.lang.numbers/Inc", value.Inc)
			// define("glojure.lang.numbers/IncP", value.IncP)
			// define("glojure.lang.Numbers/Add", value.Add)
			// define("glojure.lang.Numbers/Sub", value.Sub)
			// define("glojure.lang.Numbers/Max", value.Max)
			// define("glojure.lang.Numbers/Min", value.Min)
			// define("glojure.lang.Numbers/LT", value.LT)
		}
		// iteration functions
		{
			define("glojure.lang.iteration/NewIterator", value.NewIterator)
			define("glojure.lang.NewRangeIterator", value.NewRangeIterator)

			define("glojure.lang.iteration/NewConcatIterator", value.NewConcatIterator)

			define("glojure.lang/Pop", value.Pop)
			define("glojure.lang/Peek", value.Peek)
		}
		{ // random utilities
			define("glojure.lang.util/Compare", util.Compare)
			define("glojure.lang.util/Seq", util.Seq)
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
		{
			define("glojure.lang.BigInt", reflect.TypeOf(&value.BigInt{}))
			define("glojure.lang.PersistentHashMap", reflect.TypeOf(&value.Map{}))   // TODO: this is a hack
			define("glojure.lang.PersistentHashSet", reflect.TypeOf(&value.Set{}))   // TODO: this is a hack
			define("glojure.lang.PersistentVector", reflect.TypeOf(&value.Vector{})) // TODO: this is a hack
		}

		define("error", reflect.TypeOf((*error)(nil)).Elem())

		define("__debugstr", value.ToString)
	}
	{ // core functions
		define("glojure.lang.NewList", value.NewList)
		define("glojure.lang.CreatePersistentHashMap", value.CreatePersistentHashMap)
		define("glojure.lang.CreatePersistentTreeMap", value.CreatePersistentTreeMap)

		define("glojure.lang.Symbol", reflect.TypeOf(value.NewSymbol("")))
		define("glojure.lang.Ratio", reflect.TypeOf(value.NewRatio(1, 1)))
		define("glojure.lang.Fn", reflect.TypeOf(&value.Fn{}))
		define("glojure.lang.HasType", func(t reflect.Type, v interface{}) bool {
			if v == nil {
				return false
			}
			switch {
			case reflect.TypeOf(v) == t, reflect.TypeOf(v).ConvertibleTo(t):
				return true
			default:
				return false
			}
		})
		define("glojure.lang.TypeOf", func(v interface{}) reflect.Type {
			return reflect.TypeOf(v)
		})
		define("glojure.lang.WithMeta", value.WithMeta)
		define("glojure.lang.NewCons", value.NewCons)
		define("glojure.lang.NewSymbol", value.NewSymbol)
		define("glojure.lang.NewVector", value.NewVector)
		define("glojure.lang.NewVectorFromCollection", value.NewVectorFromCollection)
		define("glojure.lang.NewLazySeq", value.NewLazySeq)
		define("glojure.lang.NewMultiFn", value.NewMultiFn)
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
			// TODO: implement me?
		})

		define("glojure.lang.Count", value.Count)
		define("glojure.lang.Seq", value.Seq)
		define("glojure.lang.Conj", value.Conj)
		define("glojure.lang.Assoc", value.Assoc)
		define("glojure.lang.Subvec", value.Subvec)
		define("glojure.lang.First", value.First)
		define("glojure.lang.Next", value.Next)
		define("glojure.lang.Rest", value.Rest)
		define("glojure.lang.Equal", value.Equal)
		define("glojure.lang.ToString", value.ToString)
		define("glojure.lang.ToStr", value.ToStr)
		define("glojure.lang.Identical", value.Identical)
		define("glojure.lang.Get", value.Get)
		define("glojure.lang.Keys", value.Keys)
		define("glojure.lang.Vals", value.Vals)
		define("glojure.lang.GetDefault", value.GetDefault)
		define("glojure.lang.ConcatStrings", value.ConcatStrings)
		define("glojure.lang.IReduceInit", reflect.TypeOf((*value.IReduceInit)(nil)).Elem())
		define("glojure.lang.IReduce", reflect.TypeOf((*value.IReduce)(nil)).Elem())
		define("glojure.lang.IPersistentMap", reflect.TypeOf((*value.IPersistentMap)(nil)).Elem())
		define("glojure.lang.IPersistentSet", reflect.TypeOf((*value.IPersistentSet)(nil)).Elem())
		define("glojure.lang.IPersistentList", reflect.TypeOf((*value.IPersistentList)(nil)).Elem())
		define("glojure.lang.IPersistentVector", reflect.TypeOf((*value.IPersistentVector)(nil)).Elem())
		define("glojure.lang.IPersistentCollection", reflect.TypeOf((*value.IPersistentCollection)(nil)).Elem())
		define("glojure.lang.IEditableCollection", reflect.TypeOf((*value.IEditableCollection)(nil)).Elem())
		define("glojure.lang.IMeta", reflect.TypeOf((*value.IMeta)(nil)).Elem())
		define("glojure.lang.IChunkedSeq", reflect.TypeOf((*value.IChunkedSeq)(nil)).Elem())
		define("glojure.lang.ISeq", reflect.TypeOf((*value.ISeq)(nil)).Elem())
		define("glojure.lang.IDeref", reflect.TypeOf((*value.IDeref)(nil)).Elem())
		define("glojure.lang.IRecord", reflect.TypeOf((*value.IRecord)(nil)).Elem())
		define("glojure.lang.Sequential", reflect.TypeOf((*value.Sequential)(nil)).Elem())
		define("glojure.lang.IObj", reflect.TypeOf((*value.IObj)(nil)).Elem())

		define("glojure.lang.MultiFn", reflect.TypeOf(&value.MultiFn{}))
		define("glojure.lang.Namespace", reflect.TypeOf(&value.Namespace{}))

		define("glojure.lang.LockingTransaction", value.LockingTransaction)

		define("glojure.lang.Hash", value.Hash)

		define("glojure.lang.AsInt64", value.AsInt64)
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
		define("glojure.lang.CharAt", func(s string, idx int) value.Char {
			return value.NewChar([]rune(s)[idx])
		})
	}
	{
		// Add stdlib
		evalFile := func(path string) {
			core, err := stdlib.StdLib.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("could not read stdlib core.glj: %v", err))
			}
			r := reader.New(strings.NewReader(string(core)), reader.WithFilename("glojure/core.glj"), reader.WithGetCurrentNS(func() string {
				return env.CurrentNamespace().Name().String()
			}))

			for {
				expr, err := r.ReadOne()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					panic(fmt.Sprintf("error reading core lib: %v", err))
				}
				_, err = env.Eval(expr)
				if err != nil {
					panic(fmt.Sprintf("error evaluating core lib: %v", err))
				}
			}
		}
		evalFile("glojure/core.glj")
		evalFile("glojure/core_print.glj")
		evalFile("glojure/string.glj")
		evalFile("glojure/walk.glj")
		evalFile("glojure/template.glj")
		evalFile("glojure/test.glj")
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
