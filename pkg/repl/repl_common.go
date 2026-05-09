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
	if o.nreplClient == nil {
		_, err := o.env.Eval(lang.NewList(
			lang.NewSymbol("ns"),
			lang.NewSymbol(o.namespace),
		))
		if err != nil {
			panic(err)
		}
	}
	return o
}

func defaultPrompt(o *options) string {
	return o.env.CurrentNamespace().Name().String() + "=> "
}

func printBanner(w io.Writer, serverURL string) {
	noBanner := os.Getenv("GLJ_REPL_NO_BANNER")
	if noBanner == "all" {
		return
	}
	goVersion := strings.TrimPrefix(goruntime.Version(), "go")
	if noBanner == "" {
		fmt.Fprintf(w, " Glojure: %s\n", runtime.Version)
	}
	fmt.Fprintf(w, "      Go: %s %s/%s\n", goVersion, goruntime.GOOS, goruntime.GOARCH)
	if serverURL != "" {
		fmt.Fprintf(w, "  Server: %s\n", serverURL)
	}
	fmt.Fprintf(w, "    Help: C-h or :repl/help\n")
	fmt.Fprintf(w, "    Exit: C-d or :repl/exit\n")
	fmt.Fprintln(w)
}

// isBalanced returns true if parentheses, brackets, and braces are balanced.
// Used in nREPL client mode where we don't have a local reader.
func isBalanced(input string) bool {
	depth := 0
	inString := false
	escape := false
	for _, r := range input {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inString {
			escape = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
	}
	return depth <= 0 && !inString
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
	return false, false
}

// helpColors holds ANSI color codes for help output.
// CLI sets these to real ANSI codes; WASM leaves them empty.
type helpColors struct {
	BoldYellow string
	Cyan       string
	Green      string
	Reset      string
}

// noColors is used by WASM (no ANSI support).
var noColors = helpColors{}

// printHelp prints the REPL help text. editorMode is "vi" or "emacs",
// formatCmd is the current format command (e.g. "cat"),
// serverURL is the nREPL server URL (empty if no server).
func printHelp(w io.Writer, editorMode, formatCmd, serverURL string, c helpColors) {
	isEmacs := editorMode == "emacs"
	docKey := "C-d"
	helpKey := "C-h"
	printKey := "C-p"
	if isEmacs {
		docKey = "C-x C-d"
		helpKey = "C-x C-h"
		printKey = "C-x C-p"
	}
	fmt.Fprintf(w, "%sKey Bindings%s\n", c.BoldYellow, c.Reset)
	if !isEmacs {
		fmt.Fprintf(w, "  %sEscape%s    Vi normal mode; dismiss hint\n", c.Cyan, c.Reset)
	}
	fmt.Fprintf(w, "  %sTab%s       Complete symbol or insert 2-space indent\n", c.Cyan, c.Reset)
	fmt.Fprintf(w, "  %s%-10s%sShow documentation for symbol under cursor\n", c.Cyan, docKey, c.Reset)
	fmt.Fprintf(w, "  %s%-10s%sFormat, print and clipboard\n", c.Cyan, printKey, c.Reset)
	fmt.Fprintf(w, "  %sC-r%s       Reverse history search\n", c.Cyan, c.Reset)
	fmt.Fprintf(w, "  %sC-z%s       Suspend (resume with fg)\n", c.Cyan, c.Reset)
	fmt.Fprintf(w, "  %sC-c%s       Cancel input; press twice to exit\n", c.Cyan, c.Reset)
	fmt.Fprintf(w, "  %sC-d%s       Exit (on empty prompt)\n", c.Cyan, c.Reset)
	fmt.Fprintf(w, "  %s%-10s%sShow this help\n", c.Cyan, helpKey, c.Reset)
	fmt.Fprintf(w, "%sCommands%s\n", c.BoldYellow, c.Reset)
	fmt.Fprintf(w, "  %s:repl/help%s       Show this help\n", c.Green, c.Reset)
	fmt.Fprintf(w, "  %s:repl/vi%s         Switch to vi editing mode\n", c.Green, c.Reset)
	fmt.Fprintf(w, "  %s:repl/emacs%s      Switch to emacs editing mode\n", c.Green, c.Reset)
	fmt.Fprintf(w, "  %s:repl/fmt cmd%s    Set format command (for C-p)\n", c.Green, c.Reset)
	fmt.Fprintf(w, "  %s:repl/server%s     Show nREPL server URL\n", c.Green, c.Reset)
	fmt.Fprintf(w, "  %s:repl/exit%s       Exit the REPL\n", c.Green, c.Reset)
	fmt.Fprintf(w, "%sCurrent Settings%s\n", c.BoldYellow, c.Reset)
	fmt.Fprintf(w, "  %sEditor%s    %s mode\n", c.Cyan, c.Reset, editorMode)
	fmt.Fprintf(w, "  %sFormat%s    %s\n", c.Cyan, c.Reset, formatCmd)
	if serverURL != "" {
		fmt.Fprintf(w, "  %sServer%s    %s\n", c.Cyan, c.Reset, serverURL)
	}
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
