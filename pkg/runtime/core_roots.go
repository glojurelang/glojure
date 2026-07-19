package runtime

import (
	"sync"

	"github.com/glojurelang/glojure/pkg/lang"
)

var defaultCoreRoots struct {
	sync.RWMutex
	byVar map[*lang.Var]*lang.VarRootVersion
}

func recordDefaultCoreRoots(core *lang.Namespace, names ...string) {
	roots := make(map[*lang.Var]*lang.VarRootVersion, len(names))
	for _, name := range names {
		vr := core.FindInternedVar(lang.NewSymbol(name))
		if vr != nil && vr.IsBound() {
			roots[vr] = vr.RootVersion()
		}
	}
	defaultCoreRoots.Lock()
	defaultCoreRoots.byVar = roots
	defaultCoreRoots.Unlock()
}

// IsDefaultCoreVar reports whether vr still has the standard root installed
// during runtime initialization. Generated optimizations use this guard before
// replacing higher-order core operations whose Vars remain redefinable.
func IsDefaultCoreVar(vr *lang.Var) bool {
	defaultCoreRoots.RLock()
	version := defaultCoreRoots.byVar[vr]
	defaultCoreRoots.RUnlock()
	return version != nil && vr.RootVersion() == version &&
		!vr.IsMacro() && !vr.IsDynamic()
}
