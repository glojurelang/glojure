package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
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
	RT.Load("clojure/core")

	// Workaround to ensure namespaces that are required by core are loaded.
	// TODO: AOT should identify this dependency and generate code to load it.
	if useAot {
		RT.Load("clojure/core/protocols")
		RT.Load("clojure/string")
		RT.Load("glojure/go/io")
	}

	// Set the glojure version
	core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
	versionVar := core.FindInternedVar(lang.NewSymbol("*glojure-version*"))
	if versionVar != nil {
		versionVar.BindRoot(ParseVersion(VERSION))
	}

	lang.InternVar(core, lang.NewSymbol("load-file"), func(filename string) any {
		buf, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}

		kvs := make([]any, 0, 3)
		for _, vr := range []*lang.Var{lang.VarCurrentNS, lang.VarWarnOnReflection, lang.VarUncheckedMath, lang.VarDataReaders} {
			kvs = append(kvs, vr, vr.Deref())
		}
		lang.PushThreadBindings(lang.NewMap(kvs...))
		defer lang.PopThreadBindings()

		return ReadEval(string(buf), WithFilename(filename))
	}, true)

	return env
}
