package pkgmap

import (
	"reflect"
	"strings"
	"sync"
)

var (
	pkgMap = map[string]interface{}{}
	// TODO: lock-free map
	mtx sync.RWMutex

	hostClassPkg    = map[string]string{}
	hostClassPkgMtx sync.RWMutex

	hostClassType    = map[string]reflect.Type{}
	hostClassTypeMtx sync.RWMutex
)

// Set sets the value of the given package and export name.
func Set(export string, value interface{}) {
	pkg, name := SplitExport(export)

	mtx.Lock()
	defer mtx.Unlock()

	pkgMap[mungePkg(pkg)+"."+name] = value
}

// Get returns the value of the given package and export name and
// whether it was found.
func Get(export string) (interface{}, bool) {
	pkg, name := SplitExport(export)

	mtx.RLock()
	defer mtx.RUnlock()

	v, ok := pkgMap[mungePkg(pkg)+"."+name]
	return v, ok
}

func SplitExport(export string) (string, string) {
	lastDot := strings.LastIndex(export, ".")
	if lastDot == -1 {
		return "", export
	}
	pkg := export[:lastDot]
	name := export[lastDot+1:]
	return pkg, name
}

func mungePkg(pkg string) string {
	return strings.Replace(pkg, "/", ":", -1)
}

func UnmungePkg(pkg string) string {
	return strings.Replace(pkg, ":", "/", -1)
}

// HostClasses returns the set of pseudo-class pkg names that have at
// least one entry, where a "host class" is identified by a pkg part
// that is a bare identifier (e.g. "Math", "System") rather than a Go
// import path. Import paths munge to colon-separated strings and
// usually contain a dot (e.g. "github.com:..."), so this excludes any
// pkg containing either a colon or a dot. Used by REPL tab completion
// to discover javacompat host-class symbols.
func HostClasses() []string {
	mtx.RLock()
	defer mtx.RUnlock()
	seen := map[string]struct{}{}
	for key := range pkgMap {
		i := strings.LastIndexByte(key, '.')
		if i <= 0 {
			continue
		}
		pkg := key[:i]
		if strings.ContainsAny(pkg, ":.") {
			continue
		}
		seen[pkg] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// SetHostClassPackage records the Java package a host class lives in
// (e.g. "java.lang" for Math, "java.time" for Instant). Bridges call
// this once per class in init() so the REPL can offer the correct
// fully-qualified name during tab completion. If a class is not
// registered, callers should fall back to "java.lang".
func SetHostClassPackage(class, javaPkg string) {
	hostClassPkgMtx.Lock()
	defer hostClassPkgMtx.Unlock()
	hostClassPkg[class] = javaPkg
}

// HostClassPackage returns the Java package registered for the given
// host class, or "java.lang" if none has been set.
func HostClassPackage(class string) string {
	hostClassPkgMtx.RLock()
	defer hostClassPkgMtx.RUnlock()
	if p, ok := hostClassPkg[class]; ok {
		return p
	}
	return "java.lang"
}

// HostClassPackages returns a copy of the registered (class, javaPkg)
// map. Used by the REPL to enumerate fully-qualified namespace prefixes
// such as "java.time." or "java.util.regex." in tab completion.
func HostClassPackages() map[string]string {
	hostClassPkgMtx.RLock()
	defer hostClassPkgMtx.RUnlock()
	out := make(map[string]string, len(hostClassPkg))
	for k, v := range hostClassPkg {
		out[k] = v
	}
	return out
}

// SetHostClass records the reflect.Type that represents a host class.
// Bridges call this once per class in init() so namespaces can seed
// their import maps with `<short-name> -> <type>` entries, mirroring
// real Clojure's auto-import of java.lang.* (plus any other registered
// packages). The same value is also exported under the bare class name
// (e.g. "Integer") and, if the class's Java package has been registered
// via SetHostClassPackage, under the fully-qualified Java name (e.g.
// "java.lang.Integer"), so EvalASTMaybeClass can resolve both forms
// from any namespace whose mappings predate the bridge's init (notably
// clojure.core).
func SetHostClass(class string, t reflect.Type) {
	hostClassTypeMtx.Lock()
	hostClassType[class] = t
	hostClassTypeMtx.Unlock()

	Set(class, t)

	hostClassPkgMtx.RLock()
	javaPkg, ok := hostClassPkg[class]
	hostClassPkgMtx.RUnlock()
	if ok {
		Set(javaPkg+"."+class, t)
	}
}

// HostClass returns the reflect.Type registered for the given class
// and whether it was found.
func HostClass(class string) (reflect.Type, bool) {
	hostClassTypeMtx.RLock()
	defer hostClassTypeMtx.RUnlock()
	t, ok := hostClassType[class]
	return t, ok
}

// HostClassTypes returns a copy of the registered (class, type) map.
// Used by lang.NewNamespace to seed each fresh namespace with the
// auto-imported host classes so (ns-imports *ns*) is non-empty.
func HostClassTypes() map[string]reflect.Type {
	hostClassTypeMtx.RLock()
	defer hostClassTypeMtx.RUnlock()
	out := make(map[string]reflect.Type, len(hostClassType))
	for k, v := range hostClassType {
		out[k] = v
	}
	return out
}

// PkgEntries returns the names registered under the given pkg. For
// example, PkgEntries("Math") returns the JVM-style names exposed by
// the javacompat math bridge ("abs", "sqrt", "PI", ...). Returns nil
// if the pkg has no entries.
func PkgEntries(pkg string) []string {
	prefix := mungePkg(pkg) + "."
	mtx.RLock()
	defer mtx.RUnlock()
	var names []string
	for key := range pkgMap {
		if strings.HasPrefix(key, prefix) {
			names = append(names, key[len(prefix):])
		}
	}
	return names
}
