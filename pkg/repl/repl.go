//go:build !wasm

package repl

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/nrepl"
	"github.com/gloathub/glojure/pkg/pkgmap"
	"github.com/gloathub/glojure/pkg/runtime"
	"github.com/gloathub/glojure/pkg/srepl"

	// pprof
	"net/http"
	_ "net/http/pprof"
)

func init() {
	// start pprof
	if debugMode {
		go func() {
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				fmt.Println("pprof start failed:", err)
			}
		}()
	}
}

// Start starts the REPL.
func serverURL(srv *nrepl.Server) string {
	if srv == nil {
		return ""
	}
	host := srv.Host()
	if host == "0.0.0.0" || host == "::" || host == "127.0.0.1" || host == "::1" {
		host = "localhost"
	}
	return fmt.Sprintf("nrepl://%s:%d", host, srv.Port())
}

func Start(opts ...Option) {
	o := initOptions(opts)

	// Start embedded nREPL server and connect to it as a client.
	// This ensures the REPL experience is identical whether you run
	// gloat --repl (embedded server) or gloat --repl=PORT (remote).
	if o.nreplClient == nil {
		srv, err := nrepl.Start("localhost", 0, "")
		if err == nil {
			o.nreplServer = srv
			go srv.Serve()
			defer srv.Stop()

			client, err := nrepl.Connect("localhost", srv.Port())
			if err == nil {
				o.nreplClient = client
				defer client.Close()
			}
		}
	}

	// Start embedded socket REPL server (skip when connecting to external).
	sreplURL := ""
	if o.nreplServer != nil {
		sreplSrv, err := srepl.Start("localhost", 0, "")
		if err == nil {
			go sreplSrv.Serve()
			defer sreplSrv.Stop()
			h := sreplSrv.Host()
			if h == "0.0.0.0" || h == "::" || h == "127.0.0.1" || h == "::1" {
				h = "localhost"
			}
			sreplURL = fmt.Sprintf("%s:%d", h, sreplSrv.Port())
		}
	}

	rl := readline.NewShell()
	rl.Config.Vars["enable-bracketed-paste"] = true
	rl.Config.Vars["menu-complete-display-prefix"] = true

	rl.SyntaxHighlighter = func(line []rune) string {
		return highlightSyntax(line, o.env)
	}

	// Ghost text: show the common prefix of all matching completions.
	rl.SuggestFunc = func(line []rune) (suggestion []rune) {
		defer func() { recover() }()

		if len(line) == 0 {
			return nil
		}
		// Extract symbol prefix at end of line.
		end := len(line)
		start := end
		for start > 0 && isSymbolChar(line[start-1]) {
			start--
		}
		prefix := string(line[start:end])
		if prefix == "" {
			return nil
		}

		var matches []string

		if o.nreplClient != nil {
			// Client mode: use nREPL completions.
			// For qualified symbols (ns/sym), split and search the namespace.
			if i := strings.IndexByte(prefix, '/'); i >= 0 {
				nsName := prefix[:i]
				symPrefix := prefix[i+1:]
				// Ask server for completions of the symbol part within the namespace.
				entries, err := o.nreplClient.Completions(symPrefix, nsName)
				if err == nil {
					for _, e := range entries {
						matches = append(matches, strings.TrimPrefix(e.Candidate, symPrefix))
					}
				}
			} else {
				entries, err := o.nreplClient.Completions(prefix, "")
				if err == nil {
					for _, e := range entries {
						matches = append(matches, strings.TrimPrefix(e.Candidate, prefix))
					}
				}
			}
		} else if strings.HasPrefix(prefix, ":") {
			// Keyword ghost text
			kwPrefix := prefix[1:]
			for _, kw := range lang.AllKeywords() {
				if strings.HasPrefix(kw, kwPrefix) {
					matches = append(matches, kw[len(kwPrefix):])
				}
			}
		} else if i := strings.IndexByte(prefix, '/'); i >= 0 {
			// Qualified symbol (ns/prefix)
			nsName := prefix[:i]
			symPrefix := prefix[i+1:]
			ns := o.env.CurrentNamespace()
			aliasSym := lang.NewSymbol(nsName)
			targetNS := ns.LookupAlias(aliasSym)
			if targetNS == nil {
				targetNS = lang.FindNamespace(lang.NewSymbol(nsName))
			}
			if targetNS != nil {
				mappings := targetNS.Mappings()
				for seq := lang.Seq(lang.Keys(mappings)); seq != nil; seq = seq.Next() {
					name := seq.First().(*lang.Symbol).Name()
					if strings.HasPrefix(name, symPrefix) {
						matches = append(matches, name[len(symPrefix):])
					}
				}
			}
		} else {
			// Unqualified symbol
			ns := o.env.CurrentNamespace()
			mappings := ns.Mappings()
			for seq := lang.Seq(lang.Keys(mappings)); seq != nil; seq = seq.Next() {
				name := seq.First().(*lang.Symbol).Name()
				if strings.HasPrefix(name, prefix) {
					matches = append(matches, name[len(prefix):])
				}
			}
			for seq := lang.Seq(lang.Keys(ns.Aliases())); seq != nil; seq = seq.Next() {
				name := seq.First().(*lang.Symbol).Name()
				if strings.HasPrefix(name, prefix) {
					matches = append(matches, name[len(prefix):]+"/")
				}
			}
			for seq := lang.Seq(lang.AllNamespaces()); seq != nil; seq = seq.Next() {
				name := seq.First().(*lang.Namespace).Name().Name()
				if strings.HasPrefix(name, prefix) {
					matches = append(matches, name[len(prefix):]+"/")
				}
			}
		}

		if len(matches) == 0 {
			return nil
		}
		suffix := commonPrefix(matches)
		if suffix == "" {
			return nil
		}
		result := make([]rune, len(line))
		copy(result, line)
		return append(result, []rune(suffix)...)
	}

	// Bind Tab to menu-complete so all completions are shown in a menu
	tabKey := inputrc.Unescape(`\C-i`)
	for _, km := range rl.Config.Binds {
		km[tabKey] = inputrc.Bind{Action: "menu-complete"}
	}

	// Ctrl-Z: suspend the process (like a normal shell).
	ts := saveTermState()
	hintActive := false
	printText := ""
	formatCmd := os.Getenv("GLJ_REPL_FORMATTER")
	showTrace := false
	if formatCmd == "" {
		formatCmd = "cat"
	}

	// Override editing mode from env var
	if editor := os.Getenv("GLJ_REPL_EDITOR"); editor != "" {
		switch editor {
		case "vi":
			rl.Keymap.SetMain("vi-insert")
		case "emacs":
			rl.Keymap.SetMain("emacs")
		}
	}

	// Wrap vi-movement-mode: if doc hint is showing, just clear it
	// and stay in insert mode instead of switching to normal mode.
	origViMovementMode := rl.Keymap.Commands()["vi-movement-mode"]
	rl.Keymap.Register(map[string]func(){
		"vi-movement-mode": func() {
			if hintActive {
				rl.Hint.Reset()
				hintActive = false
				return
			}
			origViMovementMode()
		},
		"show-doc": func() {
			defer func() { recover() }()
			line := *rl.Line()
			pos := rl.Cursor().Pos()
			// On empty line, fall back to vi-eof-maybe (exit)
			if len(line) == 0 {
				if cmd := rl.Keymap.Commands()["vi-eof-maybe"]; cmd != nil {
					cmd()
				}
				return
			}
			// Clamp pos to valid range
			if pos >= len(line) {
				pos = len(line) - 1
			}
			// Find symbol boundaries around cursor
			start, end := pos, pos
			for start > 0 && isSymbolChar(line[start-1]) {
				start--
			}
			for end < len(line) && isSymbolChar(line[end]) {
				end++
			}
			if start == end {
				return
			}
			sym := string(line[start:end])
			// Skip numbers and other non-symbol tokens
			if isNumber(sym) || sym == "" {
				return
			}

			if o.nreplClient != nil {
				showDocRemote(o, rl, sym, &hintActive)
				return
			}

			// Resolve the symbol to a var
			ns := o.env.CurrentNamespace()
			var v *lang.Var
			if i := strings.IndexByte(sym, '/'); i >= 0 {
				nsName := sym[:i]
				symName := sym[i+1:]
				aliasSym := lang.NewSymbol(nsName)
				targetNS := ns.LookupAlias(aliasSym)
				if targetNS == nil {
					targetNS = lang.FindNamespace(lang.NewSymbol(nsName))
				}
				if targetNS != nil {
					v, _ = targetNS.Mappings().ValAt(lang.NewSymbol(symName)).(*lang.Var)
				}
			} else {
				v, _ = ns.Mappings().ValAt(lang.NewSymbol(sym)).(*lang.Var)
			}
			if v == nil {
				return
			}
			meta := v.Meta()
			if meta == nil {
				return
			}
			qualName := v.Namespace().Name().String() + "/" + v.Symbol().Name()
			var buf strings.Builder
			// ClojureDocs URL
			nsName := v.Namespace().Name().String()
			if strings.HasPrefix(nsName, "clojure.") {
				symName := v.Symbol().Name()
				urlName := strings.ReplaceAll(symName, "?", "_q")
				urlName = strings.ReplaceAll(urlName, "!", "_e")
				urlName = strings.ReplaceAll(urlName, "*", "_star")
				buf.WriteString(colorCyan)
				buf.WriteString("https://clojuredocs.org/clojure.core/" + urlName)
				buf.WriteString(colorReset)
				buf.WriteString("\r\n")
			}
			// Qualified name
			buf.WriteString(colorBoldYellow)
			buf.WriteString(qualName)
			buf.WriteString(colorReset)
			buf.WriteString("\r\n")
			// Arglists
			if arglists := meta.ValAt(lang.KWArglists); arglists != nil {
				buf.WriteString(colorGreen)
				buf.WriteString(lang.PrintString(arglists))
				buf.WriteString(colorReset)
				buf.WriteString("\r\n")
			}
			// Docstring
			if doc := meta.ValAt(lang.KWDoc); doc != nil {
				if docStr, ok := doc.(string); ok && docStr != "" {
					buf.WriteString("  ")
					buf.WriteString(strings.ReplaceAll(docStr, "\n", "\r\n  "))
				}
			}
			rl.Hint.SetTemporary(buf.String())
			hintActive = true
		},
		"show-help": func() {
			isEmacs := strings.HasPrefix(string(rl.Keymap.Main()), "emacs")
			docKey := "C-d"
			helpKey := "C-h"
			if isEmacs {
				docKey = "C-x C-d"
				helpKey = "C-x C-h"
			}
			printKey := "C-p"
			if isEmacs {
				printKey = "C-x C-p"
			}
			help := colorBoldYellow + "Key Bindings" + colorReset + "\r\n"
			if !isEmacs {
				help += "  " + colorCyan + "Escape" + colorReset + "    Vi normal mode; dismiss hint\r\n"
			}
			help += "  " + colorCyan + "Tab" + colorReset + "       Complete symbol or insert 2-space indent\r\n" +
				"  " + colorCyan + docKey + colorReset + strings.Repeat(" ", 10-len(docKey)) + "Show documentation for symbol under cursor\r\n" +
				"  " + colorCyan + printKey + colorReset + strings.Repeat(" ", 10-len(printKey)) + "Format, print and clipboard\r\n" +
				"  " + colorCyan + "C-r" + colorReset + "       Reverse history search\r\n" +
				"  " + colorCyan + "C-z" + colorReset + "       Suspend (resume with fg)\r\n" +
				"  " + colorCyan + "C-c" + colorReset + "       Cancel input; press twice to exit\r\n" +
				"  " + colorCyan + "C-d" + colorReset + "       Exit (on empty prompt)\r\n" +
				"  " + colorCyan + helpKey + colorReset + strings.Repeat(" ", 10-len(helpKey)) + "Show this help\r\n"
			help += colorBoldYellow + "Commands" + colorReset + "\r\n" +
				"  " + colorGreen + ":repl/help" + colorReset + "       Show this help\r\n" +
				"  " + colorGreen + ":repl/vi" + colorReset + "         Switch to vi editing mode\r\n" +
				"  " + colorGreen + ":repl/emacs" + colorReset + "      Switch to emacs editing mode\r\n" +
				"  " + colorGreen + ":repl/fmt cmd" + colorReset + "    Set format command (for C-p)\r\n" +
				"  " + colorGreen + ":repl/server" + colorReset + "     Show server URLs\r\n" +
				"  " + colorGreen + ":repl/show-trace" + colorReset + " Toggle panic stack traces\r\n" +
				"  " + colorGreen + ":repl/exit" + colorReset + "       Exit the REPL\r\n" +
				colorBoldYellow + "Current Settings" + colorReset + "\r\n" +
				"  " + colorCyan + "Editor" + colorReset + "    " + func() string {
				if isEmacs {
					return "emacs"
				}
				return "vi"
			}() + " mode\r\n" +
				"  " + colorCyan + "Format" + colorReset + "    " + formatCmd
			if surl := serverURL(o.nreplServer); surl != "" {
				help += "\r\n  " + colorCyan + "nREPL" + colorReset + "     " + surl
			}
			if sreplURL != "" {
				help += "\r\n  " + colorCyan + "sREPL" + colorReset + "     " + sreplURL
			}
			rl.Hint.SetTemporary(help)
			hintActive = true
		},
		"show-print": func() {
			line := rl.Line()
			if line == nil || line.Len() == 0 {
				return
			}
			raw := strings.TrimRight(string(*line), " \t\n")
			printText = runFormat(formatCmd, raw)
			copyToClipboard(printText)
			// Replace buffer with formatted text so history saves it.
			rl.Display.AcceptLine()
			line.Set([]rune(printText)...)
			rl.History.Accept(false, false, nil)
		},
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
			suspendProcess(ts, func() string { return defaultPrompt(&o) })
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
	// Bind C-d, C-h, C-p in all vi keymaps
	ctrlD := inputrc.Unescape(`\C-d`)
	ctrlH := inputrc.Unescape(`\C-h`)
	ctrlP := inputrc.Unescape(`\C-p`)
	for _, viKm := range []string{"vi", "vi-move", "vi-command", "vi-insert"} {
		if km := rl.Config.Binds[viKm]; km != nil {
			km[ctrlD] = inputrc.Bind{Action: "show-doc"}
			km[ctrlH] = inputrc.Bind{Action: "show-help"}
			km[ctrlP] = inputrc.Bind{Action: "show-print"}
		}
	}
	// Bind C-x C-d, C-x C-h, C-x C-p in emacs mode (multi-key
	// sequences in the main emacs keymap)
	if km := rl.Config.Binds["emacs"]; km != nil {
		cxcd := inputrc.Unescape(`\C-x\C-d`)
		cxch := inputrc.Unescape(`\C-x\C-h`)
		cxcp := inputrc.Unescape(`\C-x\C-p`)
		km[cxcd] = inputrc.Bind{Action: "show-doc"}
		km[cxch] = inputrc.Bind{Action: "show-help"}
		km[cxcp] = inputrc.Bind{Action: "show-print"}
	}

	rl.Prompt.Primary(func() string {
		if o.nreplClient != nil {
			return o.nreplClient.NS() + "=> "
		}
		return defaultPrompt(&o)
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

		// Cursor not at end: insert a newline (unless in vi command mode,
		// where Enter should always submit)
		viCmd := string(rl.Keymap.Main()) == "vi-command"
		if !viCmd && rl.Cursor().Pos() < rl.Line().Len() {
			return false
		}

		if o.nreplClient != nil {
			return isBalanced(input)
		}
		return isExpressionComplete(input, o.env)
	}

	// Tab completion for symbols, namespaces, and aliases
	rl.Completer = func(line []rune, cursor int) readline.Completions {
		if o.nreplClient != nil {
			return completeRemote(o, line, cursor)
		}
		return completeSymbol(o, line, cursor)
	}

	// File-based history
	histFile := o.historyFile
	if histFile == "" {
		histFile = historyFilePath()
	}
	if o.historyFmt == "jline" {
		rl.History.AddFromJLineFile("glj", histFile)
	} else {
		rl.History.AddFromFile("glj", histFile)
	}

	printBanner(o.stdout, serverURL(o.nreplServer), sreplURL)

	ctrlCPressed := false

	// evalFn wraps evalSafe with SIGINT interrupt handling.
	evalFn := func(o *options, val interface{}) (string, error) {
		return evalWithInterrupt(*o, val)
	}

	for {
		var line string
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					if showTrace {
						err = fmt.Errorf("readline panic: %v\n%s", r, debug.Stack())
					} else {
						err = fmt.Errorf("readline panic: %v", r)
					}
				}
			}()
			line, err = rl.Readline()
		}()
		if err != nil {
			if strings.HasPrefix(err.Error(), "readline panic:") {
				fmt.Fprintf(o.stdout, "PANIC: %s\n", err)
				continue
			}
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

		// Switch back to vi insert mode after submitting (only if in a vi mode)
		if m := string(rl.Keymap.Main()); m == "vi-command" || m == "vi-move" || m == "vi" {
			rl.Keymap.SetMain("vi-insert")
		}

		// show-print: print formatted text instead of evaluating
		if printText != "" {
			fmt.Fprintln(o.stdout, printText)
			printText = ""
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// CLI-specific commands (checked first so they can override shared ones)
		if trimmed == ":repl/help" {
			editorMode := "vi"
			if strings.HasPrefix(string(rl.Keymap.Main()), "emacs") {
				editorMode = "emacs"
			}
			printHelp(o.stdout, editorMode, formatCmd, serverURL(o.nreplServer), sreplURL, helpColors{
				BoldYellow: colorBoldYellow,
				Cyan:       colorCyan,
				Green:      colorGreen,
				Reset:      colorReset,
			})
			continue
		}
		if trimmed == ":repl/vi" {
			rl.Keymap.SetMain("vi-insert")
			fmt.Fprintln(o.stdout, "Switched to vi mode")
			continue
		}
		if trimmed == ":repl/emacs" {
			rl.Keymap.SetMain("emacs")
			fmt.Fprintln(o.stdout, "Switched to emacs mode")
			continue
		}
		if trimmed == ":repl/fmt" || strings.HasPrefix(trimmed, ":repl/fmt ") {
			arg := strings.TrimPrefix(trimmed, ":repl/fmt")
			arg = strings.TrimSpace(arg)
			if arg == "" {
				fmt.Fprintf(o.stdout, "Format command: %s\n", formatCmd)
			} else {
				formatCmd = arg
				fmt.Fprintf(o.stdout, "Format command set to: %s\n", formatCmd)
			}
			continue
		}
		if trimmed == ":repl/server" {
			if o.nreplServer != nil {
				url := serverURL(o.nreplServer)
				fmt.Fprintf(o.stdout, "nREPL: %s\n", url)
			}
			if sreplURL != "" {
				fmt.Fprintf(o.stdout, "sREPL: %s\n", sreplURL)
			}
			if o.nreplServer == nil && sreplURL == "" {
				fmt.Fprintln(o.stdout, "No servers running")
			}
			continue
		}
		if trimmed == ":repl/show-trace" || strings.HasPrefix(trimmed, ":repl/show-trace ") {
			arg := strings.TrimPrefix(trimmed, ":repl/show-trace")
			arg = strings.TrimSpace(arg)
			switch arg {
			case "":
				fmt.Fprintf(o.stdout, "show-trace: %v\n", showTrace)
			case "true":
				showTrace = true
				fmt.Fprintln(o.stdout, "Stack traces enabled")
			case "false":
				showTrace = false
				fmt.Fprintln(o.stdout, "Stack traces disabled")
			default:
				fmt.Fprintln(o.stdout, "Usage: :repl/show-trace [true|false]")
			}
			continue
		}

		// Shared commands (:repl/exit, etc.)
		handled, exit := handleReplCommand(trimmed, &o)
		if exit {
			return
		}
		if handled {
			continue
		}

		if o.nreplClient != nil {
			value, _, out, evalErr := o.nreplClient.Eval(line)
			if out != "" {
				fmt.Fprint(o.stdout, out)
			}
			if evalErr != nil {
				fmt.Fprintln(o.stdout, evalErr)
			} else if value != "" {
				fmt.Fprintln(o.stdout, value)
			}
		} else {
			readEvalPrint(line, &o, evalFn)
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

	sigCh, stopSig := notifyInterrupt()
	defer stopSig()

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

// ANSI color codes for syntax highlighting.
const (
	colorReset      = "\x1b[0m"
	colorGreen      = "\x1b[32m"
	colorCyan       = "\x1b[36m"
	colorMagenta    = "\x1b[35m"
	colorBlue       = "\x1b[38;5;69m"
	colorBoldYellow = "\x1b[1;33m"
	colorGray       = "\x1b[90m"
)

// Rainbow parentheses colors (Calva-style), cycling through depth levels.
var rainbowColors = []string{
	"\x1b[38;2;204;204;204m", // light gray (#ccc)
	"\x1b[38;2;0;152;230m",   // blue (#0098e6)
	"\x1b[38;2;225;109;109m", // salmon (#e16d6d)
	"\x1b[38;2;63;164;85m",   // green (#3fa455)
	"\x1b[38;2;201;104;230m", // purple (#c968e6)
	"\x1b[38;2;153;153;153m", // gray (#999)
	"\x1b[38;2;206;126;0m",   // orange (#ce7e00)
}

// Style for mismatched/unmatched closing brackets: white on red background.
const colorMismatch = "\x1b[97;41m"

// specialForms is the set of Clojure special forms and commonly
// highlighted macros, used for bold-yellow highlighting.
var specialForms = map[string]bool{
	"def": true, "defn": true, "defn-": true, "defmacro": true,
	"defonce": true, "defmethod": true, "defmulti": true,
	"defprotocol": true, "defrecord": true, "deftype": true,
	"defstruct": true, "fn": true, "fn*": true, "let": true,
	"let*": true, "loop": true, "recur": true, "if": true,
	"if-let": true, "if-not": true, "when": true, "when-let": true,
	"when-not": true, "when-first": true, "cond": true,
	"condp": true, "case": true, "do": true, "quote": true,
	"var": true, "try": true, "catch": true, "finally": true,
	"throw": true, "ns": true, "require": true, "import": true,
	"use": true, "refer": true, "in-ns": true, "for": true,
	"doseq": true, "dotimes": true, "while": true, "binding": true,
	"with-open": true, "with-local-vars": true,
}

// isClojureCoreSym returns true if the symbol resolves to a var in a
// clojure.* namespace within the given environment.
func isClojureCoreSym(env lang.Environment, token string) (result bool) {
	if env == nil {
		return false
	}
	defer func() {
		if recover() != nil {
			result = false
		}
	}()
	sym := lang.NewSymbol(token)
	if symNS := sym.Namespace(); symNS != "" {
		// Qualified symbol: look up the namespace directly.
		ns := lang.FindNamespace(lang.NewSymbol(symNS))
		if ns == nil {
			// Try as alias in current namespace.
			ns = env.CurrentNamespace().LookupAlias(lang.NewSymbol(symNS))
		}
		if ns == nil {
			return false
		}
		nsName := ns.Name().Name()
		return strings.HasPrefix(nsName, "clojure.")
	}
	// Unqualified symbol: check current namespace mappings.
	ns := env.CurrentNamespace()
	v, ok := ns.Mappings().ValAt(sym).(*lang.Var)
	if !ok {
		return false
	}
	nsName := v.Namespace().Name().Name()
	return strings.HasPrefix(nsName, "clojure.")
}

// highlightSyntax returns an ANSI-colored version of the input line
// for Clojure syntax highlighting.
func highlightSyntax(line []rune, env lang.Environment) string {
	var buf strings.Builder
	buf.Grow(len(line) * 2)
	i := 0
	n := len(line)
	var bracketStack []rune // tracks open bracket types for matching

	for i < n {
		ch := line[i]

		// String literal
		if ch == '"' {
			buf.WriteString(colorGreen)
			buf.WriteRune(ch)
			i++
			for i < n {
				c := line[i]
				buf.WriteRune(c)
				if c == '\\' && i+1 < n {
					i++
					buf.WriteRune(line[i])
				} else if c == '"' {
					break
				}
				i++
			}
			buf.WriteString(colorReset)
			i++
			continue
		}

		// Comment
		if ch == ';' {
			buf.WriteString(colorGray)
			for i < n && line[i] != '\n' {
				buf.WriteRune(line[i])
				i++
			}
			buf.WriteString(colorReset)
			continue
		}

		// Keyword
		if ch == ':' && (i == 0 || !isSymbolChar(line[i-1])) {
			start := i
			i++ // skip ':'
			if i < n && line[i] == ':' {
				i++ // skip second ':' for ::keyword
			}
			for i < n && isSymbolChar(line[i]) {
				i++
			}
			buf.WriteString(colorCyan)
			buf.WriteString(string(line[start:i]))
			buf.WriteString(colorReset)
			continue
		}

		// Opening brackets: rainbow color at current depth, then push
		if ch == '(' || ch == '[' || ch == '{' {
			depth := len(bracketStack)
			buf.WriteString(rainbowColors[depth%len(rainbowColors)])
			buf.WriteRune(ch)
			buf.WriteString(colorReset)
			bracketStack = append(bracketStack, ch)
			i++
			continue
		}

		// Closing brackets: check type match, pop, and color
		if ch == ')' || ch == ']' || ch == '}' {
			depth := len(bracketStack)
			if depth == 0 {
				// Unmatched closer
				buf.WriteString(colorMismatch)
			} else {
				open := bracketStack[depth-1]
				matched := (open == '(' && ch == ')') ||
					(open == '[' && ch == ']') ||
					(open == '{' && ch == '}')
				if matched {
					bracketStack = bracketStack[:depth-1]
					buf.WriteString(rainbowColors[(depth-1)%len(rainbowColors)])
				} else {
					// Type mismatch
					buf.WriteString(colorMismatch)
				}
			}
			buf.WriteRune(ch)
			buf.WriteString(colorReset)
			i++
			continue
		}

		// Dispatch: #, characters, deref @, quote ', etc. -- pass through
		if ch == '\'' || ch == '`' ||
			ch == '@' || ch == '^' || ch == '~' || ch == '#' {
			buf.WriteRune(ch)
			i++
			continue
		}

		// Whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ',' {
			buf.WriteRune(ch)
			i++
			continue
		}

		// Symbol or number token
		start := i
		for i < n && isSymbolChar(line[i]) {
			i++
		}
		if i == start {
			// Single non-symbol character, pass through
			buf.WriteRune(ch)
			i++
			continue
		}

		token := string(line[start:i])

		// Booleans and nil
		if token == "true" || token == "false" || token == "nil" {
			buf.WriteString(colorMagenta)
			buf.WriteString(token)
			buf.WriteString(colorReset)
			continue
		}

		// Special forms
		if specialForms[token] {
			buf.WriteString(colorBoldYellow)
			buf.WriteString(token)
			buf.WriteString(colorReset)
			continue
		}

		// Number: starts with digit, or starts with - followed by digit
		if isNumber(token) {
			buf.WriteString(colorMagenta)
			buf.WriteString(token)
			buf.WriteString(colorReset)
			continue
		}

		// clojure.* core symbol
		if isClojureCoreSym(env, token) {
			buf.WriteString(colorBlue)
			buf.WriteString(token)
			buf.WriteString(colorReset)
			continue
		}

		// Regular symbol -- no color
		buf.WriteString(token)
	}

	return buf.String()
}

func copyToClipboard(text string) {
	// Try clipboard commands in order of preference.
	for _, name := range []string{"xclip", "xsel", "wl-copy", "pbcopy"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		var cmd *exec.Cmd
		switch name {
		case "xclip":
			cmd = exec.Command(path, "-selection", "clipboard")
		case "xsel":
			cmd = exec.Command(path, "--clipboard", "--input")
		default:
			cmd = exec.Command(path)
		}
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return
		}
	}
}

func runFormat(cmdStr, text string) string {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	if err != nil {
		return text
	}
	return strings.TrimRight(string(out), "\n")
}

func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".glj_history"
	}
	return filepath.Join(home, ".glj_history")
}

// completeRemote provides tab completion via the nREPL server.
func completeRemote(o options, line []rune, cursor int) readline.Completions {
	start := cursor
	for start > 0 && isSymbolChar(line[start-1]) {
		start--
	}
	prefix := string(line[start:cursor])

	if prefix == "" {
		// Insert two spaces for indentation (same as local completer).
		if len(line) > 0 && (cursor == 0 || line[cursor-1] == ' ' || line[cursor-1] == '\t' || line[cursor-1] == '\n') {
			comps := readline.CompleteRaw([]readline.Completion{
				{Value: "  "},
			})
			comps.PREFIX = ""
			return comps
		}
		return readline.CompleteValues()
	}

	// REPL command completion
	if strings.HasPrefix(prefix, ":repl/") {
		replCmds := []string{":repl/exit", ":repl/fmt", ":repl/help", ":repl/server", ":repl/show-trace", ":repl/vi", ":repl/emacs"}
		var comps []readline.Completion
		for _, cmd := range replCmds {
			if strings.HasPrefix(cmd, prefix) {
				comps = append(comps, readline.Completion{
					Value:       cmd,
					Display:     cmd,
					Description: "repl",
				})
			}
		}
		result := readline.CompleteRaw(comps)
		result.PREFIX = prefix
		return result
	}

	// Keyword completion with :repl/ candidate
	if strings.HasPrefix(prefix, ":") {
		kwPrefix := prefix[1:]
		var comps []readline.Completion
		if strings.HasPrefix("repl/", kwPrefix) {
			comps = append(comps, readline.Completion{
				Value:       ":repl/",
				Display:     ":repl/",
				Description: "repl",
			})
		}
		entries, err := o.nreplClient.Completions(prefix, "")
		if err == nil {
			for _, e := range entries {
				comps = append(comps, readline.Completion{
					Value:       e.Candidate,
					Display:     e.Candidate,
					Description: e.NS,
				})
			}
		}
		result := readline.CompleteRaw(comps)
		result.PREFIX = prefix
		return result
	}

	entries, err := o.nreplClient.Completions(prefix, "")
	if err != nil || len(entries) == 0 {
		return readline.CompleteValues()
	}

	comps := make([]readline.Completion, len(entries))
	for i, e := range entries {
		desc := e.NS
		if e.Type == "namespace" {
			desc = "namespace"
		}
		comps[i] = readline.Completion{
			Value:       e.Candidate,
			Display:     e.Candidate,
			Description: desc,
		}
	}
	result := readline.CompleteRaw(comps)
	result.PREFIX = prefix
	return result
}

// showDocRemote fetches documentation for a symbol via nREPL eval
// and displays it as a hint.
func showDocRemote(o options, rl *readline.Shell, sym string, hintActive *bool) {
	// Use pr-str on meta to get a machine-readable result, then
	// format it ourselves. But simpler: just eval a doc-like expression
	// that returns a string we can display.
	code := fmt.Sprintf(`(let [v (resolve '%s)]
  (when v
    (let [m (meta v)
          ns-name (.Name (.Namespace v))
          sym-name (.Name (.Symbol v))]
      (str ns-name "/" sym-name "\n"
        (when-let [al (:arglists m)] (str al "\n"))
        (when (:macro m) "Macro\n")
        (when-let [d (:doc m)] (str "  " d))))))`, sym)
	value, _, _, err := o.nreplClient.Eval(code)
	if err != nil || value == "" || value == "nil" {
		return
	}
	// value is a quoted string like "clojure.core/map\n..."
	// Unescape it (it comes back as a pr-str'd string with quotes).
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
		value = strings.ReplaceAll(value, `\"`, `"`)
		value = strings.ReplaceAll(value, `\\`, `\`)
	}

	lines := strings.Split(value, "\n")
	var buf strings.Builder
	for i, line := range lines {
		if i == 0 {
			// Qualified name
			buf.WriteString(colorBoldYellow)
			buf.WriteString(line)
			buf.WriteString(colorReset)
		} else if i == 1 && strings.HasPrefix(line, "(") {
			// Arglists
			buf.WriteString(colorGreen)
			buf.WriteString(line)
			buf.WriteString(colorReset)
		} else {
			buf.WriteString(line)
		}
		if i < len(lines)-1 {
			buf.WriteString("\r\n")
		}
	}
	rl.Hint.SetTemporary(buf.String())
	*hintActive = true
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

	// REPL command completion: :repl/...
	if strings.HasPrefix(prefix, ":repl/") {
		replCmds := []string{":repl/exit", ":repl/fmt", ":repl/help", ":repl/server", ":repl/show-trace", ":repl/vi", ":repl/emacs"}
		var candidates []readline.Completion
		for _, cmd := range replCmds {
			if strings.HasPrefix(cmd, prefix) {
				candidates = append(candidates, readline.Completion{
					Value:       cmd,
					Display:     cmd,
					Description: "repl",
				})
			}
		}
		comps := readline.CompleteRaw(candidates)
		comps.PREFIX = prefix
		return comps
	}

	// Keyword completion: prefix starts with ':'
	if strings.HasPrefix(prefix, ":") {
		kwPrefix := prefix[1:] // strip leading ':'
		var candidates []readline.Completion
		// Include :repl/ as a completion candidate
		if strings.HasPrefix("repl/", kwPrefix) {
			candidates = append(candidates, readline.Completion{
				Value:       ":repl/",
				Display:     ":repl/",
				Description: "repl",
			})
		}
		for _, kw := range lang.AllKeywords() {
			if strings.HasPrefix(kw, kwPrefix) {
				candidates = append(candidates, readline.Completion{
					Value:   ":" + kw,
					Display: ":" + kw,
				})
			}
		}
		comps := readline.CompleteRaw(candidates)
		comps.PREFIX = prefix
		return comps
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

	// Javacompat host classes (Math, System, ...) registered in pkgmap.
	// Also offer the fully-qualified java.lang.<Class>/ form so users who
	// type `java.lang.M<TAB>` get completions equivalent to bare `M<TAB>`.
	// Only uppercase-leading entries are offered with the java.lang.
	// prefix, since Go stdlib packages also live in HostClasses().
	for _, hc := range pkgmap.HostClasses() {
		if strings.HasPrefix(hc, prefix) {
			candidates = append(candidates, readline.Completion{
				Value:       hc + "/",
				Display:     hc + "/",
				Description: "host class",
			})
		}
		if hc == "" || hc[0] < 'A' || hc[0] > 'Z' {
			continue
		}
		if fq := "java.lang." + hc; strings.HasPrefix(fq, prefix) {
			candidates = append(candidates, readline.Completion{
				Value:       fq + "/",
				Display:     fq + "/",
				Description: "host class",
			})
		}
	}

	comps := readline.CompleteRaw(candidates)
	comps.PREFIX = prefix

	// Only suppress trailing space after '/' when namespace/alias
	// candidates are present, so that symbol completions like
	// zero? don't have their last character trimmed on space.
	for _, c := range candidates {
		if strings.HasSuffix(c.Value, "/") {
			return comps.NoSpace('/')
		}
	}
	return comps
}

// completeQualified completes symbols within a specific namespace or alias.
func completeQualified(curNS *lang.Namespace, nsName, symPrefix, insertPrefix string) readline.Completions {
	aliasSym := lang.NewSymbol(nsName)
	targetNS := curNS.LookupAlias(aliasSym)
	if targetNS == nil {
		targetNS = lang.FindNamespace(lang.NewSymbol(nsName))
	}
	if targetNS == nil {
		return completeHostClass(nsName, symPrefix, insertPrefix)
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

// completeHostClass completes members of a javacompat host class such as
// Math/sqrt or System/getenv. It walks pkgmap for entries registered under
// the bare class name and returns each as `Class/member`. Returns an empty
// Completions if no such class is registered. A fully-qualified form like
// java.lang.Math/sqrt is treated the same as Math/sqrt.
func completeHostClass(nsName, symPrefix, insertPrefix string) readline.Completions {
	lookup := nsName
	if bare, ok := strings.CutPrefix(nsName, "java.lang."); ok {
		lookup = bare
	}
	names := pkgmap.PkgEntries(lookup)
	if len(names) == 0 {
		return readline.Completions{}
	}
	var candidates []readline.Completion
	for _, name := range names {
		if !strings.HasPrefix(name, symPrefix) {
			continue
		}
		candidates = append(candidates, readline.Completion{
			Value:       insertPrefix + name,
			Display:     insertPrefix + name,
			Description: nsName,
		})
	}
	comps := readline.CompleteRaw(candidates)
	comps.PREFIX = insertPrefix + symPrefix
	return comps
}
