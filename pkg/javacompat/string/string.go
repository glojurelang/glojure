// Package string exposes JVM-faithful java.lang.String equivalents for
// code running on glojure. Each symbol is published in up to three ways:
//
//   - as a Go package-level value (used when gloat AOT-compiles a Clojure
//     call site to a direct Go reference such as `compatstring.Format`);
//   - through glojure's pkgmap under both `String.foo` and the fully
//     qualified `github.com/gloathub/glojure/pkg/javacompat/string.Foo`
//     names (used by the REPL and any dynamic resolution path); and
//   - for instance-style methods (`(.toUpperCase s)`, `(.length s)`,
//     etc.) via lang.RegisterStringMethod, which the FieldOrMethod
//     dispatch path consults when the receiver is a Go string.
//
// Where the JVM signature is overloaded by arity (indexOf(ch) /
// indexOf(ch, from) / indexOf(sub) / indexOf(sub, from)), the bridge
// dispatches polymorphically at call time. Char-typed args accept either
// int32 (UTF-16 unit) or lang.Char.
package string

import (
	"fmt"
	"reflect"

	jstr "github.com/gloathub/gojava/string"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/string"

// Static methods exposed as Go package-level functions for AOT use.

func Format(args ...any) string {
	if len(args) == 0 {
		panic("String/format: missing format string")
	}
	f := toString(args[0])
	return jstr.Format(f, args[1:])
}

func Join(args ...any) string {
	if len(args) < 1 {
		panic("String/join: missing delimiter")
	}
	delim := toString(args[0])
	if len(args) == 2 {
		if parts, ok := toStringSlice(args[1]); ok {
			return jstr.Join(delim, parts)
		}
	}
	rest := make([]string, 0, len(args)-1)
	for _, a := range args[1:] {
		rest = append(rest, toString(a))
	}
	return jstr.Join(delim, rest)
}

func ValueOf(x any) string { return jstr.ValueOf(x) }

func CopyValueOf(args ...any) string {
	data, ok := toInt32Slice(args[0])
	if !ok {
		panic(fmt.Sprintf("String/copyValueOf: cannot coerce %T to char[]", args[0]))
	}
	switch len(args) {
	case 1:
		return jstr.CopyValueOf(data, 0, int32(len(data)))
	case 3:
		return jstr.CopyValueOf(data, toInt32(args[1]), toInt32(args[2]))
	}
	panic(fmt.Sprintf("String/copyValueOf: wrong number of args (%d)", len(args)))
}

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("String."+jvmName, v)
	pkgmap.SetHostClassPackage("String", "java.lang")
	pkgmap.SetHostClass("String", reflect.TypeOf(""))
}

func registerMethod(name string, fn lang.StringMethod) {
	lang.RegisterStringMethod(name, fn)
}

func init() {
	// Static methods. Registered under bare "String." prefix; the runtime
	// also accepts "java.lang.String/" via prefix stripping in evalast.go.
	register("format", "Format", lang.FnFunc(func(args ...any) any { return Format(args...) }))
	register("join", "Join", lang.FnFunc(func(args ...any) any { return Join(args...) }))
	register("valueOf", "ValueOf", lang.FnFunc(func(args ...any) any { return ValueOf(args[0]) }))
	register("copyValueOf", "CopyValueOf", lang.FnFunc(func(args ...any) any { return CopyValueOf(args...) }))

	// Instance methods. The lang.FieldOrMethod path captures the receiver
	// and invokes these with the remaining call args.
	registerMethod("length", func(s string, _ ...any) any { return jstr.Length(s) })
	registerMethod("isEmpty", func(s string, _ ...any) any { return jstr.IsEmpty(s) })
	registerMethod("isBlank", func(s string, _ ...any) any { return jstr.IsBlank(s) })
	registerMethod("charAt", func(s string, rest ...any) any {
		return lang.NewChar(rune(jstr.CharAt(s, toInt32(rest[0]))))
	})
	registerMethod("codePointAt", func(s string, rest ...any) any {
		return jstr.CodePointAt(s, toInt32(rest[0]))
	})
	registerMethod("indexOf", func(s string, rest ...any) any {
		return indexOfDispatch(s, rest, false)
	})
	registerMethod("lastIndexOf", func(s string, rest ...any) any {
		return indexOfDispatch(s, rest, true)
	})
	registerMethod("substring", func(s string, rest ...any) any {
		switch len(rest) {
		case 1:
			return jstr.Substring(s, toInt32(rest[0]))
		case 2:
			return jstr.SubstringRange(s, toInt32(rest[0]), toInt32(rest[1]))
		}
		panic(fmt.Sprintf("String/substring: wrong number of args (%d)", len(rest)))
	})
	registerMethod("toUpperCase", func(s string, _ ...any) any { return jstr.ToUpperCase(s) })
	registerMethod("toLowerCase", func(s string, _ ...any) any { return jstr.ToLowerCase(s) })
	registerMethod("trim", func(s string, _ ...any) any { return jstr.Trim(s) })
	registerMethod("strip", func(s string, _ ...any) any { return jstr.Strip(s) })
	registerMethod("stripLeading", func(s string, _ ...any) any { return jstr.StripLeading(s) })
	registerMethod("stripTrailing", func(s string, _ ...any) any { return jstr.StripTrailing(s) })
	registerMethod("startsWith", func(s string, rest ...any) any {
		switch len(rest) {
		case 1:
			return jstr.StartsWith(s, toString(rest[0]))
		case 2:
			return jstr.StartsWithOffset(s, toString(rest[0]), toInt32(rest[1]))
		}
		panic(fmt.Sprintf("String/startsWith: wrong number of args (%d)", len(rest)))
	})
	registerMethod("endsWith", func(s string, rest ...any) any {
		return jstr.EndsWith(s, toString(rest[0]))
	})
	registerMethod("contains", func(s string, rest ...any) any {
		return jstr.Contains(s, toString(rest[0]))
	})
	registerMethod("equals", func(s string, rest ...any) any {
		if t, ok := rest[0].(string); ok {
			return jstr.Equals(s, t)
		}
		return false
	})
	registerMethod("equalsIgnoreCase", func(s string, rest ...any) any {
		if t, ok := rest[0].(string); ok {
			return jstr.EqualsIgnoreCase(s, t)
		}
		return false
	})
	registerMethod("compareTo", func(s string, rest ...any) any {
		return jstr.CompareTo(s, toString(rest[0]))
	})
	registerMethod("compareToIgnoreCase", func(s string, rest ...any) any {
		return jstr.CompareToIgnoreCase(s, toString(rest[0]))
	})
	registerMethod("concat", func(s string, rest ...any) any {
		return jstr.Concat(s, toString(rest[0]))
	})
	registerMethod("repeat", func(s string, rest ...any) any {
		return jstr.Repeat(s, toInt32(rest[0]))
	})
	registerMethod("replace", func(s string, rest ...any) any {
		a, b := rest[0], rest[1]
		if _, ok := a.(string); ok {
			return jstr.Replace(s, toString(a), toString(b))
		}
		if _, ok := b.(string); ok {
			return jstr.Replace(s, toString(a), toString(b))
		}
		return jstr.ReplaceChar(s, toInt32(a), toInt32(b))
	})
	registerMethod("replaceAll", func(s string, rest ...any) any {
		out, err := jstr.ReplaceAll(s, toString(rest[0]), toString(rest[1]))
		if err != nil {
			panic(err.Error())
		}
		return out
	})
	registerMethod("replaceFirst", func(s string, rest ...any) any {
		out, err := jstr.ReplaceFirst(s, toString(rest[0]), toString(rest[1]))
		if err != nil {
			panic(err.Error())
		}
		return out
	})
	registerMethod("matches", func(s string, rest ...any) any {
		out, err := jstr.Matches(s, toString(rest[0]))
		if err != nil {
			panic(err.Error())
		}
		return out
	})
	registerMethod("split", func(s string, rest ...any) any {
		limit := int32(0)
		if len(rest) > 1 {
			limit = toInt32(rest[1])
		}
		parts, err := jstr.Split(s, toString(rest[0]), limit)
		if err != nil {
			panic(err.Error())
		}
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return lang.NewVector(out...)
	})
	registerMethod("toCharArray", func(s string, _ ...any) any {
		units := jstr.ToCharArray(s)
		out := make([]any, len(units))
		for i, u := range units {
			out[i] = lang.NewChar(rune(u))
		}
		return lang.NewVector(out...)
	})
	registerMethod("getBytes", func(s string, _ ...any) any { return jstr.GetBytes(s) })
	registerMethod("chars", func(s string, _ ...any) any {
		units := jstr.Chars(s)
		out := make([]any, len(units))
		for i, u := range units {
			out[i] = int32(u)
		}
		return lang.NewVector(out...)
	})
	registerMethod("codePoints", func(s string, _ ...any) any {
		cps := jstr.CodePoints(s)
		out := make([]any, len(cps))
		for i, c := range cps {
			out[i] = int32(c)
		}
		return lang.NewVector(out...)
	})
	registerMethod("lines", func(s string, _ ...any) any {
		ls := jstr.Lines(s)
		out := make([]any, len(ls))
		for i, l := range ls {
			out[i] = l
		}
		return lang.NewVector(out...)
	})
	registerMethod("intern", func(s string, _ ...any) any { return jstr.Intern(s) })
	registerMethod("hashCode", func(s string, _ ...any) any { return jstr.HashCode(s) })
	registerMethod("toString", func(s string, _ ...any) any { return jstr.ToString(s) })
}

// indexOfDispatch handles indexOf/lastIndexOf: one or two args, where the
// first arg may be a char (int32 / lang.Char) or a substring.
func indexOfDispatch(s string, rest []any, last bool) int32 {
	if len(rest) < 1 || len(rest) > 2 {
		panic(fmt.Sprintf("String/indexOf: wrong number of args (%d)", len(rest)))
	}
	if sub, ok := rest[0].(string); ok {
		if last {
			if len(rest) == 1 {
				return jstr.LastIndexOf(s, sub)
			}
			return jstr.LastIndexOfFrom(s, sub, toInt32(rest[1]))
		}
		if len(rest) == 1 {
			return jstr.IndexOf(s, sub)
		}
		return jstr.IndexOfFrom(s, sub, toInt32(rest[1]))
	}
	ch := toInt32(rest[0])
	if last {
		if len(rest) == 1 {
			return jstr.LastIndexOfChar(s, ch)
		}
		return jstr.LastIndexOfCharFrom(s, ch, toInt32(rest[1]))
	}
	if len(rest) == 1 {
		return jstr.IndexOfChar(s, ch)
	}
	return jstr.IndexOfCharFrom(s, ch, toInt32(rest[1]))
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
	case float64:
		return int32(v)
	case float32:
		return int32(v)
	case lang.Char:
		return int32(rune(v))
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
}

func toStringSlice(x any) ([]string, bool) {
	switch v := x.(type) {
	case []string:
		return v, true
	case []any:
		out := make([]string, len(v))
		for i, e := range v {
			out[i] = toString(e)
		}
		return out, true
	case lang.IPersistentVector:
		n := v.Count()
		out := make([]string, n)
		for i := 0; i < n; i++ {
			out[i] = toString(v.Nth(i))
		}
		return out, true
	case lang.ISeq:
		var out []string
		for s := v; s != nil; s = s.Next() {
			out = append(out, toString(s.First()))
		}
		return out, true
	}
	return nil, false
}

func toInt32Slice(x any) ([]int32, bool) {
	switch v := x.(type) {
	case []int32:
		return v, true
	case []any:
		out := make([]int32, len(v))
		for i, e := range v {
			out[i] = toInt32(e)
		}
		return out, true
	case lang.IPersistentVector:
		n := v.Count()
		out := make([]int32, n)
		for i := 0; i < n; i++ {
			out[i] = toInt32(v.Nth(i))
		}
		return out, true
	case lang.ISeq:
		var out []int32
		for s := v; s != nil; s = s.Next() {
			out = append(out, toInt32(s.First()))
		}
		return out, true
	}
	return nil, false
}
