// Package system exposes JVM-faithful java.lang.System equivalents for code
// running on glojure. Each symbol is published two ways:
//
//   - as a Go package-level value (used when gloat AOT-compiles a Clojure
//     call site to a direct Go reference such as `compatsystem.Exit`); and
//   - through glojure's pkgmap (used by the REPL and any dynamic
//     resolution path).
//
// Where the JVM signature returns a possibly-null String (System.getenv,
// System.getProperty), the bridge converts gojava's (string, bool) result
// into either the string value or nil so Clojure idioms like
// (when-let [v (System/getenv "HOME")] ...) work without extra glue.
package system

import (
	"fmt"
	"reflect"

	jsys "github.com/gloathub/gojava/system"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/system"

// System is the placeholder type registered as java.lang.System's
// reflect.Type. System has private constructors in the JVM, so no
// instances exist; the value just makes (ns-imports *ns*) include the
// auto-imported class.
type System struct{}

// Stream values, exposed under System.out / System.err / System.in. They
// carry the Go methods (Println, Print, Printf, Write, Flush, Read) that
// glojure's FieldOrMethod resolves to from Clojure call sites like
// (.println System/out ...) (it capitalizes the first letter).
var (
	Out = jsys.Out
	Err = jsys.Err
	In  = jsys.In
)

// CurrentTimeMillis returns ms since epoch.
func CurrentTimeMillis() int64 { return jsys.CurrentTimeMillis() }

// NanoTime returns a high-resolution time source.
func NanoTime() int64 { return jsys.NanoTime() }

// Getenv dispatches on arity: zero args returns a Clojure-visible map of all
// env vars; one arg returns the value as a string, or nil if unset.
func Getenv(args ...any) any {
	switch len(args) {
	case 0:
		env := jsys.GetenvAll()
		kvs := make([]any, 0, len(env)*2)
		for k, v := range env {
			kvs = append(kvs, k, v)
		}
		return lang.NewPersistentHashMap(kvs...)
	case 1:
		if v, ok := jsys.Getenv(toString(args[0])); ok {
			return v
		}
		return nil
	}
	panic(fmt.Sprintf("System/getenv: wrong number of args (%d)", len(args)))
}

// GetProperty dispatches on arity: one arg returns the value or nil; two
// args returns the value or the default.
func GetProperty(args ...any) any {
	switch len(args) {
	case 1:
		if v, ok := jsys.GetProperty(toString(args[0])); ok {
			return v
		}
		return nil
	case 2:
		return jsys.GetPropertyOr(toString(args[0]), toString(args[1]))
	}
	panic(fmt.Sprintf("System/getProperty: wrong number of args (%d)", len(args)))
}

// SetProperty returns the previous value (or nil if none).
func SetProperty(name, value any) any {
	old, ok := jsys.SetProperty(toString(name), toString(value))
	if !ok {
		return nil
	}
	return old
}

// ClearProperty returns the previous value (or nil if none).
func ClearProperty(name any) any {
	old, ok := jsys.ClearProperty(toString(name))
	if !ok {
		return nil
	}
	return old
}

func Exit(code any) {
	jsys.Exit(toInt32(code))
}

func LineSeparator() string { return jsys.LineSeparator() }

func Gc() { jsys.Gc() }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("System."+jvmName, v)
	pkgmap.SetHostClassPackage("System", "java.lang")
	pkgmap.SetHostClass("System", reflect.TypeOf(System{}))
}

func init() {
	register("out", "Out", Out)
	register("err", "Err", Err)
	register("in", "In", In)

	register("currentTimeMillis", "CurrentTimeMillis", lang.FnFunc(func(args ...any) any {
		return CurrentTimeMillis()
	}))
	register("nanoTime", "NanoTime", lang.FnFunc(func(args ...any) any {
		return NanoTime()
	}))

	register("getenv", "Getenv", lang.FnFunc(func(args ...any) any { return Getenv(args...) }))
	register("getProperty", "GetProperty", lang.FnFunc(func(args ...any) any { return GetProperty(args...) }))
	register("setProperty", "SetProperty", lang.FnFunc(func(args ...any) any {
		return SetProperty(args[0], args[1])
	}))
	register("clearProperty", "ClearProperty", lang.FnFunc(func(args ...any) any {
		return ClearProperty(args[0])
	}))

	register("exit", "Exit", lang.FnFunc(func(args ...any) any {
		Exit(args[0])
		return nil
	}))
	register("lineSeparator", "LineSeparator", lang.FnFunc(func(args ...any) any {
		return LineSeparator()
	}))
	register("gc", "Gc", lang.FnFunc(func(args ...any) any {
		Gc()
		return nil
	}))
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
	}
	panic(fmt.Sprintf("cannot coerce %T to int32", x))
}
