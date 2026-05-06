//go:build !wasm

package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/reeflective/readline"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/reader"
	"github.com/gloathub/glojure/pkg/runtime"

	// pprof
	"net/http"
	_ "net/http/pprof"
)

const debugMode = false
const cpuProfile = false

func init() {
	// start pprof
	if debugMode {
		go func() {
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				fmt.Println("pprof start failed:", err)
			}
		}()
		// shell command to examine pprof profile with a web ui:
		// $ go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile
	}
}

// Start starts the REPL.
func Start(opts ...Option) {
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
	{ // set namespace
		_, err := o.env.Eval(lang.NewList(
			lang.NewSymbol("ns"),
			lang.NewSymbol(o.namespace),
		))
		if err != nil {
			panic(err)
		}
	}

	defaultPrompt := func() string {
		curNS := "?"
		ns := o.env.CurrentNamespace()
		curNS = ns.Name().String()
		return curNS + "=> "
	}

	rl := readline.NewShell()
	rl.Config.Vars["enable-bracketed-paste"] = true

	rl.Prompt.Primary(func() string {
		return defaultPrompt()
	})
	rl.Prompt.Secondary(func() string {
		return "... "
	})

	// AcceptMultiline: return false if expression is incomplete (needs more input)
	rl.AcceptMultiline = func(line []rune) bool {
		input := string(line)
		if strings.TrimSpace(input) == "" {
			return true
		}
		rdr := reader.New(
			strings.NewReader(input),
			reader.WithFilename("repl"),
			reader.WithGetCurrentNS(func() *lang.Namespace {
				return o.env.CurrentNamespace()
			}),
		)
		_, err := rdr.ReadAll()
		if err != nil && errors.Is(err, io.EOF) {
			return false
		}
		return true
	}

	// File-based history
	histFile := historyFilePath()
	rl.History.AddFromFile("glj", histFile)

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				fmt.Fprint(o.stdout, "\r\n")
				continue
			}
			break
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		rdr := reader.New(strings.NewReader(line), reader.WithFilename("repl"), reader.WithGetCurrentNS(func() *lang.Namespace {
			return o.env.CurrentNamespace()
		}))

		vals, err := rdr.ReadAll()
		if err != nil {
			fmt.Fprintln(o.stdout, err)
			continue
		}

		for _, val := range vals {
			out, err := evalWithInterrupt(o, val)
			if err != nil {
				fmt.Fprintf(o.stdout, "\r\n%s\r\n", err)
				if err.Error() == "Interrupted" {
					break
				}
				continue
			}
			fmt.Fprintln(o.stdout, out)
		}
	}
}

// evalWithInterrupt runs eval in a goroutine so SIGINT can interrupt it.
// On Ctrl-C, the eval goroutine is abandoned and "Interrupted" is returned.
func evalWithInterrupt(o options, val interface{}) (string, error) {
	type result struct {
		out string
		err error
	}

	// Capture thread-local bindings (including *ns*) so the eval
	// goroutine inherits them. Go goroutines don't share thread-local
	// storage, so without this *ns* would revert to clojure.core.
	bindings := lang.GetThreadBindings()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)

	resCh := make(chan result, 1)
	go func() {
		lang.PushThreadBindings(bindings)
		defer lang.PopThreadBindings()
		defer func() {
			if panicErr := recover(); panicErr != nil {
				resCh <- result{"", fmt.Errorf("panic: %v\nstacktrace:\n%s", panicErr, string(debug.Stack()))}
			}
		}()
		v, err := o.env.Eval(val)
		runtime.Debug = false
		if err != nil {
			resCh <- result{"", err}
			return
		}
		resCh <- result{lang.PrintString(v), nil}
	}()

	select {
	case <-sigCh:
		return "", fmt.Errorf("Interrupted")
	case r := <-resCh:
		return r.out, r.err
	}
}

func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".glj_history"
	}
	return filepath.Join(home, ".glj_history")
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

	// TODO: clean up this code. copied from rtcompat.go.
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
