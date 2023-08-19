package pkgmap

import (
	"strings"
	"sync"
)

var (
	pkgMap = map[string]interface{}{}
	// TODO: lock-free map
	mtx sync.RWMutex
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
	return strings.Replace(pkg, "/", "$", -1)
}

func UnmungePkg(pkg string) string {
	return strings.Replace(pkg, "$", "/", -1)
}
