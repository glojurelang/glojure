// Package character exposes JVM-faithful java.lang.Character equivalents for
// code running on glojure. Glojure's character literal `\a` parses to
// lang.Char (a rune wrapper); the coercion helpers here unwrap both Char
// and plain ints so callers can pass either form.
package character

import (
	"fmt"
	"reflect"

	jchar "github.com/gloathub/gojava/character"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/character"

const (
	MIN_RADIX = jchar.MIN_RADIX
	MAX_RADIX = jchar.MAX_RADIX
)

var (
	MIN_VALUE = lang.Char(jchar.MIN_VALUE)
	MAX_VALUE = lang.Char(jchar.MAX_VALUE)
)

// ValueOf returns its char argument unchanged (as lang.Char so the value
// keeps its glojure-visible Character identity, matching the (Character. c)
// constructor sugar).
func ValueOf(x any) lang.Char { return lang.Char(toRune(x)) }

func IsDigit(x any) bool          { return jchar.IsDigit(toRune(x)) }
func IsLetter(x any) bool         { return jchar.IsLetter(toRune(x)) }
func IsLetterOrDigit(x any) bool  { return jchar.IsLetterOrDigit(toRune(x)) }
func IsAlphabetic(x any) bool     { return jchar.IsAlphabetic(toRune(x)) }
func IsWhitespace(x any) bool     { return jchar.IsWhitespace(toRune(x)) }
func IsSpaceChar(x any) bool      { return jchar.IsSpaceChar(toRune(x)) }
func IsUpperCase(x any) bool      { return jchar.IsUpperCase(toRune(x)) }
func IsLowerCase(x any) bool      { return jchar.IsLowerCase(toRune(x)) }

func ToUpperCase(x any) lang.Char { return lang.Char(jchar.ToUpperCase(toRune(x))) }
func ToLowerCase(x any) lang.Char { return lang.Char(jchar.ToLowerCase(toRune(x))) }
func ToString(x any) string       { return jchar.ToString(toRune(x)) }

func Digit(args ...any) any {
	if len(args) != 2 {
		panic(fmt.Sprintf("Character/digit: wrong number of args (%d)", len(args)))
	}
	return jchar.Digit(toRune(args[0]), int(toInt32(args[1])))
}

func ForDigit(args ...any) any {
	if len(args) != 2 {
		panic(fmt.Sprintf("Character/forDigit: wrong number of args (%d)", len(args)))
	}
	return lang.Char(jchar.ForDigit(int(toInt32(args[0])), int(toInt32(args[1]))))
}

func GetNumericValue(x any) int32 { return jchar.GetNumericValue(toRune(x)) }

func Compare(x, y any) int32 { return jchar.Compare(toRune(x), toRune(y)) }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Character."+jvmName, v)
	pkgmap.SetHostClassPackage("Character", "java.lang")
	pkgmap.SetHostClass("Character", reflect.TypeOf(lang.Char(0)))
}

func init() {
	register("MIN_VALUE", "MIN_VALUE", MIN_VALUE)
	register("MAX_VALUE", "MAX_VALUE", MAX_VALUE)
	register("MIN_RADIX", "MIN_RADIX", int32(MIN_RADIX))
	register("MAX_RADIX", "MAX_RADIX", int32(MAX_RADIX))

	register("valueOf", "ValueOf", lang.FnFunc(func(args ...any) any { return ValueOf(args[0]) }))

	register("isDigit", "IsDigit", lang.FnFunc(func(args ...any) any { return IsDigit(args[0]) }))
	register("isLetter", "IsLetter", lang.FnFunc(func(args ...any) any { return IsLetter(args[0]) }))
	register("isLetterOrDigit", "IsLetterOrDigit", lang.FnFunc(func(args ...any) any { return IsLetterOrDigit(args[0]) }))
	register("isAlphabetic", "IsAlphabetic", lang.FnFunc(func(args ...any) any { return IsAlphabetic(args[0]) }))
	register("isWhitespace", "IsWhitespace", lang.FnFunc(func(args ...any) any { return IsWhitespace(args[0]) }))
	register("isSpaceChar", "IsSpaceChar", lang.FnFunc(func(args ...any) any { return IsSpaceChar(args[0]) }))
	register("isUpperCase", "IsUpperCase", lang.FnFunc(func(args ...any) any { return IsUpperCase(args[0]) }))
	register("isLowerCase", "IsLowerCase", lang.FnFunc(func(args ...any) any { return IsLowerCase(args[0]) }))

	register("toUpperCase", "ToUpperCase", lang.FnFunc(func(args ...any) any { return ToUpperCase(args[0]) }))
	register("toLowerCase", "ToLowerCase", lang.FnFunc(func(args ...any) any { return ToLowerCase(args[0]) }))
	register("toString", "ToString", lang.FnFunc(func(args ...any) any { return ToString(args[0]) }))

	register("digit", "Digit", lang.FnFunc(func(args ...any) any { return Digit(args...) }))
	register("forDigit", "ForDigit", lang.FnFunc(func(args ...any) any { return ForDigit(args...) }))
	register("getNumericValue", "GetNumericValue", lang.FnFunc(func(args ...any) any { return GetNumericValue(args[0]) }))

	register("compare", "Compare", lang.FnFunc(func(args ...any) any { return Compare(args[0], args[1]) }))
}

func toRune(x any) rune {
	switch v := x.(type) {
	case lang.Char:
		return rune(v)
	case int32: // also matches rune
		return v
	case int64:
		return rune(v)
	case int:
		return rune(v)
	case uint8:
		return rune(v)
	case uint16:
		return rune(v)
	case string:
		if len(v) == 0 {
			panic("cannot coerce empty string to char")
		}
		for _, r := range v {
			return r
		}
	}
	panic(fmt.Sprintf("cannot coerce %T to rune", x))
}

func toInt32(x any) int32 {
	switch v := x.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case lang.Char:
		return int32(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
}
