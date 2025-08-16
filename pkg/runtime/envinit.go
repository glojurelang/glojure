package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/stdlib"
)

// The current version of Glojure
const VERSION = "0.3.0"

// ParseVersion parses the VERSION string and returns a map with major, minor,
// incremental, and qualifier
func ParseVersion(version string) lang.IPersistentMap {
	parts := strings.Split(version, ".")

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	incremental := 0
	qualifier := interface{}(nil)

	if len(parts) > 2 {
		// Check if the third part contains a qualifier (e.g., "0-alpha")
		incrementalPart := parts[2]
		if strings.Contains(incrementalPart, "-") {
			qualifierParts := strings.SplitN(incrementalPart, "-", 2)
			incremental, _ = strconv.Atoi(qualifierParts[0])
			qualifier = qualifierParts[1]
		} else {
			incremental, _ = strconv.Atoi(incrementalPart)
		}
	}

	return lang.NewMap(
		lang.NewKeyword("major"), major,
		lang.NewKeyword("minor"), minor,
		lang.NewKeyword("incremental"), incremental,
		lang.NewKeyword("qualifier"), qualifier,
	)
}

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
	// TODO: this is rather rather hacky
	lang.GlobalEnv = env

	// bootstrap namespace control
	{
		// bootstrap implementation of the ns macro
		env.DefVar(lang.NewSymbol("in-ns"), lang.IFnFunc(func(args ...interface{}) interface{} {
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

	{
		// Add stdlib
		evalFile := func(path string) {
			core, err := stdlib.StdLib.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("could not read stdlib core.glj: %v", err))
			}
			r := reader.New(strings.NewReader(string(core)), reader.WithFilename(path), reader.WithGetCurrentNS(func() *lang.Namespace {
				return env.CurrentNamespace()
			}))

			for {
				expr, err := r.ReadOne()
				if err == reader.ErrEOF {
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

	// Set the glojure version
	core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
	versionVar := core.FindInternedVar(lang.NewSymbol("*glojure-version*"))
	if versionVar != nil {
		versionVar.BindRoot(ParseVersion(VERSION))
	}

	// Set the glojure load path
	loadPathVar := core.FindInternedVar(lang.NewSymbol("*glojure-load-path*"))
	if loadPathVar != nil {
		loadPathVar.BindRoot(lang.Seq(GetLoadPath()))
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
