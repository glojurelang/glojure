package runtime

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
)

// version is set via -ldflags at build time.
var version string

var (
	// The current version of Glojure
	Version = func() string {
		if version != "" {
			return strings.TrimPrefix(version, "v")
		}
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return "0.0.0"
		}
		if info.Main.Version == "" || info.Main.Version == "(devel)" {
			return "0.0.0"
		}
		return strings.TrimPrefix(info.Main.Version, "v")
	}()
)

// parseVersion parses the Version string and returns a map with major, minor,
// incremental, and qualifier
func parseVersion(version string) lang.IPersistentMap {
	parts := strings.Split(version, ".")

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	incremental := 0
	var qualifier any

	if len(parts) > 2 {
		// Check if the third part contains a qualifier (e.g., "0-alpha")
		incrementalPart := strings.Join(parts[2:], ".")
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

	// Set the glojure version
	core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
	if installNativeCoreOverrides {
		installNativeCoreFunctions(core)
	}
	versionVar := core.FindInternedVar(lang.NewSymbol("*glojure-version*"))
	if versionVar != nil {
		versionVar.BindRoot(parseVersion(Version))
	}

	// Override promise with a Go implementation since the Clojure version
	// uses java.util.concurrent.CountDownLatch that doesn't exist in Go.
	lang.InternVar(core, lang.NewSymbol("promise"), lang.FnFunc(func(args ...any) any {
		return NewPromise()
	}), true)

	// Override shuffle with a Go implementation since the Clojure version
	// uses Java interop (java.util.Collections/shuffle) that doesn't exist.
	lang.InternVar(core, lang.NewSymbol("shuffle"), lang.FnFunc(func(args ...any) any {
		coll := args[0]
		if lang.IsNil(coll) {
			panic(lang.NewIllegalArgumentError("shuffle requires a collection, got nil"))
		}
		// Only accept seqable collections, not strings or maps
		switch coll.(type) {
		case string:
			panic(lang.NewIllegalArgumentError("shuffle requires a collection, got string"))
		case lang.IPersistentMap:
			panic(lang.NewIllegalArgumentError("shuffle requires a collection, got map"))
		}
		// Convert to slice, shuffle, return vector
		var elems []any
		for s := lang.Seq(coll); s != nil; s = s.Next() {
			elems = append(elems, s.First())
		}
		// Fisher-Yates shuffle
		for i := len(elems) - 1; i > 0; i-- {
			j := int(rand.Int63n(int64(i + 1)))
			elems[i], elems[j] = elems[j], elems[i]
		}
		return lang.NewVector(elems...)
	}), true)

	lang.InternVar(core, lang.NewSymbol("add-load-path"), lang.FnFunc1(func(path any) any {
		AddLoadPath(os.DirFS(path.(string)))
		return nil
	}), true)

	lang.InternVar(core, lang.NewSymbol("load"), lang.FnFunc(func(paths ...any) any {
		if len(paths) == 0 {
			panic("Wrong number of args passed to: clojure.core/load")
		}
		for _, path := range paths {
			RT.Load(path.(string))
		}
		return nil
	}), true)

	lang.InternVar(core, lang.NewSymbol("load-file"), lang.FnFunc1(func(filename any) any {
		fname := filename.(string)
		buf, err := os.ReadFile(fname)
		if err != nil {
			panic(err)
		}

		kvs := make([]any, 0, 3)
		for _, vr := range []*lang.Var{lang.VarCurrentNS, lang.VarWarnOnReflection, lang.VarUncheckedMath, lang.VarDataReaders} {
			kvs = append(kvs, vr, vr.Deref())
		}
		lang.PushThreadBindings(lang.NewMap(kvs...))
		defer lang.PopThreadBindings()

		return ReadEval(string(buf), WithFilename(fname))
	}), true)

	return env
}
