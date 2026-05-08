package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/reader"
	"github.com/gloathub/glojure/pkg/runtime"
)

const debugMode = false
const cpuProfile = false

// EvalFunc is the signature for eval functions passed to readEvalPrint.
// CLI wraps evalSafe with signal handling; WASM uses evalSafe directly.
type EvalFunc func(*options, interface{}) (string, error)

func initOptions(opts []Option) options {
	o := options{
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		namespace: "user",
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.env == nil {
		o.env = initEnv(o.stdout)
	}
	_, err := o.env.Eval(lang.NewList(
		lang.NewSymbol("ns"),
		lang.NewSymbol(o.namespace),
	))
	if err != nil {
		panic(err)
	}
	return o
}

func defaultPrompt(o *options) string {
	return o.env.CurrentNamespace().Name().String() + "=> "
}

func printBanner(w io.Writer) {
	noBanner := os.Getenv("GLJ_REPL_NO_BANNER")
	if noBanner == "all" {
		return
	}
	goVersion := strings.TrimPrefix(goruntime.Version(), "go")
	if noBanner == "" {
		fmt.Fprintf(w, " Glojure: v%s\n", runtime.Version)
	}
	fmt.Fprintf(w, "      Go: %s %s/%s\n", goVersion, goruntime.GOOS, goruntime.GOARCH)
	fmt.Fprintf(w, "    Help: C-h or :repl/help\n")
	fmt.Fprintf(w, "    Exit: C-d or :repl/exit\n")
	fmt.Fprintln(w)
}

// isExpressionComplete returns true if input parses without hitting EOF
// (i.e., all delimiters are balanced).
func isExpressionComplete(input string, env lang.Environment) bool {
	if strings.TrimSpace(input) == "" {
		return true
	}
	rdr := reader.New(
		strings.NewReader(input),
		reader.WithFilename("repl"),
		reader.WithGetCurrentNS(func() *lang.Namespace {
			return env.CurrentNamespace()
		}),
	)
	_, err := rdr.ReadAll()
	return !(err != nil && errors.Is(err, io.EOF))
}

// evalSafe evaluates a single form with panic recovery, no signal handling.
func evalSafe(o *options, val interface{}) (string, error) {
	var out string
	var err error
	func() {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				err = fmt.Errorf("panic: %v\nstacktrace:\n%s", panicErr, string(debug.Stack()))
			}
		}()
		var v interface{}
		v, err = o.env.Eval(val)
		runtime.Debug = false
		if err != nil {
			return
		}
		out = lang.PrintString(v)
	}()
	return out, err
}

// handleReplCommand processes :repl/* commands that work on all platforms.
// Returns (handled, exit). If handled is false, the caller should check
// for platform-specific commands before falling through to eval.
func handleReplCommand(trimmed string, o *options) (handled, exit bool) {
	if trimmed == ":repl/exit" {
		return true, true
	}
	if trimmed == ":repl/help" {
		fmt.Fprintln(o.stdout, "Commands")
		fmt.Fprintln(o.stdout, "  :repl/help       Show this help")
		fmt.Fprintln(o.stdout, "  :repl/exit       Exit the REPL")
		return true, false
	}
	return false, false
}

// readEvalPrint reads all forms from input, evals each with evalFn, and
// prints results or errors.
func readEvalPrint(input string, o *options, evalFn EvalFunc) {
	rdr := reader.New(
		strings.NewReader(input),
		reader.WithFilename("repl"),
		reader.WithGetCurrentNS(func() *lang.Namespace {
			return o.env.CurrentNamespace()
		}),
	)
	vals, err := rdr.ReadAll()
	if err != nil {
		fmt.Fprintln(o.stdout, err)
		return
	}
	for _, val := range vals {
		out, err := evalFn(o, val)
		if err != nil {
			fmt.Fprintln(o.stdout, err)
			continue
		}
		fmt.Fprintln(o.stdout, out)
	}
}

func initEnv(stdout io.Writer) lang.Environment {
	if cpuProfile {
		f, err := os.Create("./gljInitEnvCpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	startTime := time.Now()

	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*lang.Var{lang.VarCurrentNS, lang.VarWarnOnReflection, lang.VarUncheckedMath, lang.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	lang.PushThreadBindings(lang.NewMap(kvs...))

	env := runtime.NewEnvironment(runtime.WithStdout(stdout))
	if debugMode {
		fmt.Printf("Environment created in %v\n", time.Since(startTime))
	}

	return env
}

func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	if s[0] == '-' || s[0] == '+' {
		i++
		if i >= len(s) {
			return false
		}
	}
	if s[i] < '0' || s[i] > '9' {
		return false
	}
	return true
}

func isSymbolChar(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	return strings.ContainsRune(".*+!-_?/<>=$&%#:", r)
}

func commonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	prefix := ss[0]
	for _, s := range ss[1:] {
		for i := range prefix {
			if i >= len(s) || s[i] != prefix[i] {
				prefix = prefix[:i]
				break
			}
		}
		if len(s) < len(prefix) {
			prefix = prefix[:len(s)]
		}
	}
	return prefix
}
