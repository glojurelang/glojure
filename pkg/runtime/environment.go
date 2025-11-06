package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/glojurelang/glojure/pkg/lang"
)

var (
	SymbolUnquote       = lang.NewSymbol("clojure.core/unquote")
	SymbolSpliceUnquote = lang.NewSymbol("splice-unquote")
	SymbolNamespace     = lang.NewSymbol("ns")
	SymbolInNamespace   = lang.NewSymbol("in-ns")
	SymbolUserNamespace = lang.NewSymbol("user")
	SymbolDot           = lang.NewSymbol(".")
)

type (
	environment struct {
		ctx context.Context

		// local bindings
		scope *scope

		recurTarget interface{}

		// some well-known vars
		namespaceVar   *lang.Var // ns
		inNamespaceVar *lang.Var // in-ns

		// counter for gensym (symbol generator)
		symCounter int32

		stdout io.Writer
		stderr io.Writer

		loadPath []string
	}
)

func newEnvironment(ctx context.Context, stdout, stderr io.Writer) *environment {
	e := &environment{
		ctx:    ctx,
		scope:  newScope(),
		stdout: stdout,
		stderr: stderr,
	}
	coreNS := lang.NSCore

	for _, dyn := range []string{
		"command-line-args",
		"warn-on-reflection",
		"compile-path",
		"unchecked-math",
		"compiler-options",
		"err",
		"flush-on-newline",
		"print-meta",
		"print-dup",
		"read-eval",
		"glojure-version",
		"load-path",
	} {
		coreNS.InternWithValue(lang.NewSymbol("*"+dyn+"*"), nil, true).SetDynamic()
	}

	// bootstrap some vars
	e.namespaceVar = coreNS.InternWithValue(SymbolNamespace,
		lang.NewFnFunc(func(args ...interface{}) interface{} {
			return coreNS
		}), true)
	e.namespaceVar.SetMacro()

	e.inNamespaceVar = lang.NewVarWithRoot(coreNS, SymbolInNamespace, false)

	return e
}

func (env *environment) nextSymNum() int32 {
	for {
		val := atomic.LoadInt32(&env.symCounter)
		if atomic.CompareAndSwapInt32(&env.symCounter, val, val+1) {
			return val
		}
	}
}

func (env *environment) Context() context.Context {
	return env.ctx
}

func (env *environment) String() string {
	return fmt.Sprintf("object[Environment]")
}

// TODO: rename to something else; this isn't for `def`s, it's for
// local bindings.
func (env *environment) BindLocal(sym *lang.Symbol, val interface{}) {
	env.scope.define(sym, val)
}

func (env *environment) DefVar(sym *lang.Symbol, val interface{}) *lang.Var {
	// TODO: match clojure implementation more closely
	v := env.CurrentNamespace().InternWithValue(sym, val, true /* replace root */)
	if meta := sym.Meta(); meta != nil {
		v.SetMeta(meta)
	}
	return v
}

func (env *environment) lookup(sym *lang.Symbol) (res interface{}, ok bool) {
	v, ok := env.scope.lookup(sym)
	if ok {
		return v, true
	}

	ns := env.CurrentNamespace()
	if sym.Namespace() != "" {
		ns = lang.FindNamespace(lang.NewSymbol(sym.Namespace()))
		sym = lang.NewSymbol(sym.Name())
	}
	if ns == nil {
		return nil, false
	}
	vr := ns.Mappings().ValAt(sym)
	if vr == nil {
		return nil, false
	}
	// TODO: can these only be vars?
	return vr.(*lang.Var).Get(), true
}

func (env *environment) WithRecurTarget(rt interface{}) lang.Environment {
	wrappedEnv := *env
	newEnv := &wrappedEnv
	newEnv.recurTarget = rt
	return newEnv
}

func (env *environment) PushScope() lang.Environment {
	wrappedEnv := *env
	newEnv := &wrappedEnv
	newEnv.scope = newEnv.scope.push()
	return newEnv
}

func (env *environment) Stdout() io.Writer {
	return env.stdout
}

func (env *environment) Stderr() io.Writer {
	return env.stderr
}

func (env *environment) CurrentNamespace() *lang.Namespace {
	return lang.VarCurrentNS.Get().(*lang.Namespace)
}

func (env *environment) SetCurrentNamespace(ns *lang.Namespace) {
	lang.VarCurrentNS.Set(ns)
}

func (env *environment) PushLoadPaths(paths []string) lang.Environment {
	newEnv := &(*env)
	newEnv.loadPath = append(paths, newEnv.loadPath...)
	return newEnv
}

func (env *environment) ResolveFile(filename string) (string, bool) {
	if filepath.IsAbs(filename) {
		return filename, true
	}

	for _, path := range env.loadPath {
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, true
		}
	}
	return "", false
}

func (env *environment) Errorf(n interface{}, format string, args ...interface{}) error {
	return env.errorf(n, format, args...)
}

func (env *environment) errorf(n interface{}, format string, args ...interface{}) error {
	var filename, line, col string
	var meta lang.IPersistentMap
	if n, ok := n.(lang.IObj); ok {
		meta = n.Meta()
	}
	get := func(m lang.IPersistentMap, key string) string {
		return lang.PrintString(lang.GetDefault(m, lang.NewKeyword(key), "?"))
	}

	filename = get(meta, "file")
	line = get(meta, "line")
	col = get(meta, "column")

	location := fmt.Sprintf("%s:%s:%s", filename, line, col)

	return fmt.Errorf("%s: "+format, append([]interface{}{location}, args...)...)
}

// LookupLocal looks up a local binding in the environment.
// This is used by the codegen system to access captured values.
func (env *environment) LookupLocal(name string) (any, bool) {
	if env == nil || env.scope == nil {
		return nil, false
	}
	return env.scope.lookup(lang.NewSymbol(name))
}
