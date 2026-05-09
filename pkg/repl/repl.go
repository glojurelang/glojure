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
	"github.com/gloathub/glojure/pkg/runtime"

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

	// Start embedded nREPL server for editor connectivity (local mode only).
	if o.nreplClient == nil {
		srv, err := nrepl.Start("localhost", 0, "")
		if err == nil {
			o.nreplServer = srv
			go srv.Serve()
			defer srv.Stop()
		}
	}

	rl := readline.NewShell()
	rl.Config.Vars["enable-bracketed-paste"] = true
	rl.Config.Vars["menu-complete-display-prefix"] = true

	if o.nreplClient == nil {
		rl.SyntaxHighlighter = func(line []rune) string {
			return highlightSyntax(line, o.env)
		}
	}

	// Ghost text: show the common prefix of all matching completions.
	if o.nreplClient == nil {
		rl.SuggestFunc = func(line []rune) []rune {
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

			// Keyword ghost text
			if strings.HasPrefix(prefix, ":") {
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
			if o.nreplClient != nil {
				return
			}
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
				"  " + colorGreen + ":repl/server" + colorReset + "     Show nREPL server URL\r\n" +
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
				help += "\r\n  " + colorCyan + "Server" + colorReset + "    " + surl
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
	histFile := historyFilePath()
	rl.History.AddFromFile("glj", histFile)

	printBanner(o.stdout, serverURL(o.nreplServer))

	ctrlCPressed := false

	// evalFn wraps evalSafe with SIGINT interrupt handling.
	evalFn := func(o *options, val interface{}) (string, error) {
		return evalWithInterrupt(*o, val)
	}

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
			printHelp(o.stdout, editorMode, formatCmd, serverURL(o.nreplServer), helpColors{
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
				fmt.Fprintf(o.stdout, "nREPL server on port %d on host %s - %s\n",
					o.nreplServer.Port(), "localhost", url)
			} else {
				fmt.Fprintln(o.stdout, "No nREPL server running")
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
func isClojureCoreSym(env lang.Environment, token string) bool {
	ns := env.CurrentNamespace()
	sym := lang.NewSymbol(token)
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

		// Dispatch: #, characters, deref @, quote ', etc. -- pass through
		if ch == '(' || ch == ')' || ch == '[' || ch == ']' ||
			ch == '{' || ch == '}' || ch == '\'' || ch == '`' ||
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
		return readline.CompleteValues()
	}
	candidates, err := o.nreplClient.Completions(prefix, "")
	if err != nil || len(candidates) == 0 {
		return readline.CompleteValues()
	}
	values := make([]string, len(candidates))
	for i, c := range candidates {
		values[i] = strings.TrimPrefix(c, prefix)
	}
	return readline.CompleteValues(values...).Prefix(prefix)
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
		replCmds := []string{":repl/exit", ":repl/fmt", ":repl/help", ":repl/server", ":repl/vi", ":repl/emacs"}
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
