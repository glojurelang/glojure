// Package classes registers common JVM marker and runtime classes that map
// directly onto existing Glojure host types.
package classes

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func register(name, javaPackage string, typ reflect.Type) {
	pkgmap.SetHostClassPackage(name, javaPackage)
	pkgmap.SetHostClass(name, lang.NewClass(typ, javaPackage+"."+name))
}

func init() {
	register("Object", "java.lang", reflect.TypeOf((*any)(nil)).Elem())
	register("Comparable", "java.lang", reflect.TypeOf((*lang.Comparer)(nil)).Elem())
	register("Set", "java.util", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem())
	register("List", "java.util", reflect.TypeOf((*lang.Sequential)(nil)).Elem())
	register("Map", "java.util", reflect.TypeOf((*lang.IPersistentMap)(nil)).Elem())
	register("Keyword", "clojure.lang", reflect.TypeOf((*lang.Keyword)(nil)))
	register("IDeref", "clojure.lang", reflect.TypeOf((*lang.IDeref)(nil)).Elem())
	register("Associative", "clojure.lang", reflect.TypeOf((*lang.Associative)(nil)).Elem())
	register("Counted", "clojure.lang", reflect.TypeOf((*lang.Counted)(nil)).Elem())
	register("IAtom", "clojure.lang", reflect.TypeOf((*lang.IAtom)(nil)).Elem())
	register("IBlockingDeref", "clojure.lang", reflect.TypeOf((*lang.IBlockingDeref)(nil)).Elem())
	register("IChunkedSeq", "clojure.lang", reflect.TypeOf((*lang.IChunkedSeq)(nil)).Elem())
	register("IEditableCollection", "clojure.lang", reflect.TypeOf((*lang.IEditableCollection)(nil)).Elem())
	register("IFn", "clojure.lang", reflect.TypeOf((*lang.IFn)(nil)).Elem())
	register("ILookup", "clojure.lang", reflect.TypeOf((*lang.ILookup)(nil)).Elem())
	register("IMapEntry", "clojure.lang", reflect.TypeOf((*lang.IMapEntry)(nil)).Elem())
	register("IMeta", "clojure.lang", reflect.TypeOf((*lang.IMeta)(nil)).Elem())
	register("IObj", "clojure.lang", reflect.TypeOf((*lang.IObj)(nil)).Elem())
	register("IPending", "clojure.lang", reflect.TypeOf((*lang.IPending)(nil)).Elem())
	register("IPersistentCollection", "clojure.lang", reflect.TypeOf((*lang.IPersistentCollection)(nil)).Elem())
	register("IPersistentMap", "clojure.lang", reflect.TypeOf((*lang.IPersistentMap)(nil)).Elem())
	register("IPersistentSet", "clojure.lang", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem())
	register("IPersistentVector", "clojure.lang", reflect.TypeOf((*lang.IPersistentVector)(nil)).Elem())
	register("IRecord", "clojure.lang", reflect.TypeOf((*lang.IRecord)(nil)).Elem())
	register("IReduceInit", "clojure.lang", reflect.TypeOf((*lang.IReduceInit)(nil)).Elem())
	register("IRef", "clojure.lang", reflect.TypeOf((*lang.IRef)(nil)).Elem())
	register("ISeq", "clojure.lang", reflect.TypeOf((*lang.ISeq)(nil)).Elem())
	register("Indexed", "clojure.lang", reflect.TypeOf((*lang.Indexed)(nil)).Elem())
	register("Named", "clojure.lang", reflect.TypeOf((*lang.Named)(nil)).Elem())
	register("Reversible", "clojure.lang", reflect.TypeOf((*lang.Reversible)(nil)).Elem())
	register("Seqable", "clojure.lang", reflect.TypeOf((*lang.Seqable)(nil)).Elem())
	register("Sequential", "clojure.lang", reflect.TypeOf((*lang.Sequential)(nil)).Elem())
	register("Sorted", "clojure.lang", reflect.TypeOf((*lang.Sorted)(nil)).Elem())
	register("Ratio", "clojure.lang", reflect.TypeOf((*lang.Ratio)(nil)))
	register("Symbol", "clojure.lang", reflect.TypeOf((*lang.Symbol)(nil)))
	register("Atom", "clojure.lang", reflect.TypeOf((*lang.Atom)(nil)))
	register("Volatile", "clojure.lang", reflect.TypeOf((*lang.Volatile)(nil)))
}
