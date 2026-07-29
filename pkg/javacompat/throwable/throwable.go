// Package throwable exposes the common java.lang throwable hierarchy.
package throwable

import (
	"errors"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func New(args ...any) any {
	if len(args) == 0 {
		return errors.New("")
	}
	return errors.New(lang.ToString(args[0]))
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
