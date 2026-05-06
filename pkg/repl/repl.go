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

	"golang.org/x/sys/unix"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"

	goruntime "runtime"

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
	rl.Config.Vars["menu-complete-display-prefix"] = true

	// Bind Tab to menu-complete so all completions are shown in a menu
	tabKey := inputrc.Unescape(`\C-i`)
	for _, km := range rl.Config.Binds {
		km[tabKey] = inputrc.Bind{Action: "menu-complete"}
	}

	// Ctrl-Z: suspend the process (like a normal shell).
	fd := int(os.Stdin.Fd())
	cookedState, _ := unix.IoctlGetTermios(fd, unix.TCGETS)
	rl.Keymap.Register(map[string]func(){
		"smart-backspace": func() {
			pos := rl.Cursor().Pos()
			line := rl.Line()
			if pos >= 2 && (*line)[pos-1] == ' ' && (*line)[pos-2] == ' ' {
				rl.Cursor().Dec()
				rl.Cursor().Dec()
				line.Cut(pos-2, pos)
			} else if pos > 0 {
				rl.Cursor().Dec()
				line.CutRune(rl.Cursor().Pos())
			}
		},
		"suspend": func() {
			rawState, _ := unix.IoctlGetTermios(fd, unix.TCGETS)
			unix.IoctlSetTermios(fd, unix.TCSETS, cookedState)
			fmt.Print("\r\n")
			// Reset to default so the kernel handles SIGTSTP directly.
			// Wait for SIGCONT to know when fg has resumed us.
			contCh := make(chan os.Signal, 1)
			signal.Notify(contCh, syscall.SIGCONT)
			signal.Reset(syscall.SIGTSTP)
			syscall.Kill(syscall.Getpid(), syscall.SIGTSTP)
			<-contCh
			signal.Stop(contCh)
			// Restore raw mode and redisplay prompt
			unix.IoctlSetTermios(fd, unix.TCSETS, rawState)
			fmt.Print(defaultPrompt())
		},
	})
	backspace := inputrc.Unescape(`\C-?`)
	for _, km := range rl.Config.Binds {
		km[backspace] = inputrc.Bind{Action: "smart-backspace"}
	}
	ctrlZ := inputrc.Unescape(`\C-z`)
	for _, km := range rl.Config.Binds {
		km[ctrlZ] = inputrc.Bind{Action: "suspend"}
	}

	rl.Prompt.Primary(func() string {
		return defaultPrompt()
	})
	rl.Prompt.Secondary(func() string {
		return "... "
	})

	// AcceptMultiline: return false if expression is incomplete (needs more input).
	// Also insert a newline (instead of submitting) when the cursor is not at the
	// end of the buffer, even if the expression is already complete.
	rl.AcceptMultiline = func(line []rune) bool {
		input := string(line)
		if strings.TrimSpace(input) == "" {
			return true
		}

		// Cursor not at end: always insert a newline
		if rl.Cursor().Pos() < rl.Line().Len() {
			return false
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

	// Tab completion for symbols, namespaces, and aliases
	rl.Completer = func(line []rune, cursor int) readline.Completions {
		return completeSymbol(o, line, cursor)
	}

	// File-based history
	histFile := historyFilePath()
	rl.History.AddFromFile("glj", histFile)

	printBanner(o.stdout)

	ctrlCPressed := false

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				if line != "" {
					// Was editing: just cancel the input
					ctrlCPressed = false
					fmt.Fprintln(o.stdout)
					continue
				}
				if ctrlCPressed {
					fmt.Fprintln(o.stdout)
					return
				}
				ctrlCPressed = true
				fmt.Fprintln(o.stdout, "(To exit, press Ctrl+C again or Ctrl+D or type :repl/exit)")
				continue
			}
			break
		}

		ctrlCPressed = false

		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.TrimSpace(line) == ":repl/exit" {
			return
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

func printBanner(w io.Writer) {
	fmt.Fprintf(w, "Glojure v%s\n", runtime.Version)
	goVersion := strings.TrimPrefix(goruntime.Version(), "go")
	fmt.Fprintf(w, "Go %s %s/%s\n", goVersion, goruntime.GOOS, goruntime.GOARCH)
	fmt.Fprintf(w, "    Docs: (doc function-name)\n")
	fmt.Fprintf(w, "  Source: (source function-name)\n")
	fmt.Fprintf(w, "    Exit: Ctrl+D or :repl/exit\n")
	fmt.Fprintln(w)
}

func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".glj_history"
	}
	return filepath.Join(home, ".glj_history")
}

// completeSymbol provides tab completion for Clojure symbols.
// It handles: bare symbols, namespace-qualified symbols (ns/sym),
// alias-qualified symbols (alias/sym), and namespace names.
func completeSymbol(o options, line []rune, cursor int) readline.Completions {
	start := cursor
	for start > 0 && isSymbolChar(line[start-1]) {
		start--
	}
	prefix := string(line[start:cursor])

	if prefix == "" {
		// Insert two spaces for indentation when cursor is after
		// whitespace (or at line start) and the line isn't empty.
		if len(line) > 0 && (cursor == 0 || line[cursor-1] == ' ' || line[cursor-1] == '\t' || line[cursor-1] == '\n') {
			comps := readline.CompleteRaw([]readline.Completion{
				{Value: "  "},
			})
			comps.PREFIX = ""
			return comps
		}
		return readline.Completions{}
	}

	ns := o.env.CurrentNamespace()

	if i := strings.IndexByte(prefix, '/'); i >= 0 {
		nsPrefix := prefix[:i]
		symPrefix := prefix[i+1:]
		return completeQualified(ns, nsPrefix, symPrefix, prefix[:i+1])
	}

	var candidates []readline.Completion

	// Symbols from current namespace mappings (includes refers)
	mappings := ns.Mappings()
	for seq := lang.Seq(lang.Keys(mappings)); seq != nil; seq = seq.Next() {
		sym := seq.First().(*lang.Symbol)
		name := sym.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		desc := ""
		if v, ok := mappings.ValAt(sym).(*lang.Var); ok {
			desc = v.Namespace().Name().Name()
		}
		candidates = append(candidates, readline.Completion{
			Value:       name,
			Display:     name,
			Description: desc,
		})
	}

	// Namespace aliases
	for seq := lang.Seq(lang.Keys(ns.Aliases())); seq != nil; seq = seq.Next() {
		name := seq.First().(*lang.Symbol).Name()
		if strings.HasPrefix(name, prefix) {
			candidates = append(candidates, readline.Completion{
				Value:       name + "/",
				Display:     name + "/",
				Description: "alias",
			})
		}
	}

	// Full namespace names
	for seq := lang.Seq(lang.AllNamespaces()); seq != nil; seq = seq.Next() {
		name := seq.First().(*lang.Namespace).Name().Name()
		if strings.HasPrefix(name, prefix) {
			candidates = append(candidates, readline.Completion{
				Value:       name + "/",
				Display:     name + "/",
				Description: "namespace",
			})
		}
	}

	comps := readline.CompleteRaw(candidates)
	comps.PREFIX = prefix
	return comps.NoSpace('/')
}

// completeQualified completes symbols within a specific namespace or alias.
func completeQualified(curNS *lang.Namespace, nsName, symPrefix, insertPrefix string) readline.Completions {
	aliasSym := lang.NewSymbol(nsName)
	targetNS := curNS.LookupAlias(aliasSym)
	if targetNS == nil {
		targetNS = lang.FindNamespace(lang.NewSymbol(nsName))
	}
	if targetNS == nil {
		return readline.Completions{}
	}

	var candidates []readline.Completion
	nsDesc := targetNS.Name().Name()
	mappings := targetNS.Mappings()
	for seq := lang.Seq(lang.Keys(mappings)); seq != nil; seq = seq.Next() {
		sym := seq.First().(*lang.Symbol)
		name := sym.Name()
		if !strings.HasPrefix(name, symPrefix) {
			continue
		}
		candidates = append(candidates, readline.Completion{
			Value:       insertPrefix + name,
			Display:     insertPrefix + name,
			Description: nsDesc,
		})
	}

	comps := readline.CompleteRaw(candidates)
	comps.PREFIX = insertPrefix + symPrefix
	return comps
}

func isSymbolChar(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	return strings.ContainsRune(".*+!-_?/<>=$&%#", r)
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
