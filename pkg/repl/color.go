package repl

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
)

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

// ColorSyntax returns an ANSI-colored version of the input
// for Clojure syntax highlighting.
func ColorSyntax(line []rune, env lang.Environment) string {
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
