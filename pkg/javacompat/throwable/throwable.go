// Package throwable exposes the common java.lang throwable hierarchy.
package throwable

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func New(args ...any) any {
	switch len(args) {
	case 0:
		return errors.New("")
	case 1:
		return errors.New(lang.ToString(args[0]))
	case 2:
		cause, ok := args[1].(error)
		if !ok {
			panic(fmt.Sprintf("Throwable/new: cause must be an error, got %T", args[1]))
		}
		return fmt.Errorf("%s: %w", lang.ToString(args[0]), cause)
	default:
		panic(fmt.Sprintf("Throwable/new: wrong number of args (%d)", len(args)))
	}
}

func register(name string) {
	pkgmap.SetHostClassPackage(name, "java.lang")
	pkgmap.SetHostClass(name, lang.NewClass(errorType, "java.lang."+name))
	lang.RegisterHostConstructor("java.lang."+name,
		lang.FnFunc(func(args ...any) any { return New(args...) }))
}

func init() {
	for _, name := range []string{
		"Throwable",
		"Exception",
		"RuntimeException",
		"IllegalArgumentException",
		"IllegalStateException",
		"UnsupportedOperationException",
	} {
		register(name)
	}
}
