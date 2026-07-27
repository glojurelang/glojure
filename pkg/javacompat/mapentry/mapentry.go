// Package mapentry exposes the clojure.lang.MapEntry static factory used by
// portable Clojure tests and libraries.
package mapentry

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

const pkg = "github.com/glojurelang/glojure/pkg/javacompat/mapentry"

// Create mirrors clojure.lang.MapEntry/create.
func Create(key, value any) *lang.MapEntry {
	return lang.NewMapEntry(key, value)
}

func init() {
	pkgmap.Set(pkg+".Create", Create)
	pkgmap.Set("MapEntry.create", Create)
	pkgmap.Set("clojure.lang.MapEntry.create", Create)
	pkgmap.SetHostClassPackage("MapEntry", "clojure.lang")
	pkgmap.SetHostClass(
		"MapEntry",
		lang.NewClass(reflect.TypeOf((*lang.MapEntry)(nil)), "clojure.lang.MapEntry"),
	)
}
