package lang

import "github.com/glojurelang/glojure/pkg/pkgmap"

// Static members of clojure.lang classes that Clojure source refers to
// directly. They are registered in the pkgmap so that
// pkgmap.LookupHostMember resolves them for both the evaluator and
// AOT-compiled code.
func init() {
	pkgmap.Set("clojure.lang.PersistentTreeMap.create",
		FnFunc(func(args ...any) any {
			if len(args) == 2 {
				return CreatePersistentTreeMapWithComparator(
					args[0].(IFn), args[1])
			}
			return CreatePersistentTreeMap(args[0])
		}))
	pkgmap.Set("clojure.lang.MapEntry.create",
		FnFunc(func(args ...any) any {
			return NewMapEntry(args[0], args[1])
		}))
}
