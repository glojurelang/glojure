// Package regex exposes JVM-faithful java.util.regex equivalents for code
// running on glojure. Static methods (Pattern/compile, Pattern/matches,
// Pattern/quote) and Pattern flag constants are published through pkgmap
// under both the bare "Pattern." prefix and the fully qualified Go path.
// Instance methods on *Pattern and *Matcher (matcher, find, group, ...)
// are reached at runtime through lang.FieldOrMethod, which dispatches via
// reflection on the receiver; the gojava regex package uses capitalized,
// variadic method signatures so the JVM's overloaded forms collapse to a
// single Go method per name.
package regex

import (
	"fmt"

	jregex "github.com/gloathub/gojava/regex"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/regex"

const (
	CASE_INSENSITIVE = jregex.CASE_INSENSITIVE
	MULTILINE        = jregex.MULTILINE
	LITERAL          = jregex.LITERAL
	DOTALL           = jregex.DOTALL
	UNICODE_CASE     = jregex.UNICODE_CASE
)

// Compile mirrors Pattern.compile(String). Single-arg form; the
// flag-aware overload is reached via CompileFlags.
func Compile(args ...any) *jregex.Pattern {
	if len(args) == 0 {
		panic("Pattern/compile: missing regex string")
	}
	if len(args) == 1 {
		p, err := jregex.Compile(toString(args[0]))
		if err != nil {
			panic(err.Error())
		}
		return p
	}
	p, err := jregex.CompileFlags(toString(args[0]), toInt32(args[1]))
	if err != nil {
		panic(err.Error())
	}
	return p
}

// Matches mirrors Pattern.matches(regex, input): convenience for a
// one-shot whole-input match.
func Matches(args ...any) bool {
	if len(args) != 2 {
		panic(fmt.Sprintf("Pattern/matches: wrong number of args (%d)", len(args)))
	}
	out, err := jregex.Matches(toString(args[0]), toString(args[1]))
	if err != nil {
		panic(err.Error())
	}
	return out
}

// Quote mirrors Pattern.quote: returns a literal pattern.
func Quote(args ...any) string {
	if len(args) != 1 {
		panic(fmt.Sprintf("Pattern/quote: wrong number of args (%d)", len(args)))
	}
	return jregex.Quote(toString(args[0]))
}

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Pattern."+jvmName, v)
	pkgmap.SetHostClassPackage("Pattern", "java.util.regex")
}

func init() {
	register("CASE_INSENSITIVE", "CASE_INSENSITIVE", int32(CASE_INSENSITIVE))
	register("MULTILINE", "MULTILINE", int32(MULTILINE))
	register("LITERAL", "LITERAL", int32(LITERAL))
	register("DOTALL", "DOTALL", int32(DOTALL))
	register("UNICODE_CASE", "UNICODE_CASE", int32(UNICODE_CASE))

	register("compile", "Compile", lang.FnFunc(func(args ...any) any { return Compile(args...) }))
	register("matches", "Matches", lang.FnFunc(func(args ...any) any { return Matches(args...) }))
	register("quote", "Quote", lang.FnFunc(func(args ...any) any { return Quote(args...) }))
}

func toString(x any) string {
	switch v := x.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func toInt32(x any) int32 {
	switch v := x.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case int16:
		return int32(v)
	case int8:
		return int32(v)
	case uint32:
		return int32(v)
	case uint16:
		return int32(v)
	case uint8:
		return int32(v)
	case lang.Char:
		return int32(rune(v))
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
}
