package runtime

import (
	"sync/atomic"

	"github.com/glojurelang/glojure/pkg/lang"
)

type coreRootTable struct {
	byVar map[*lang.Var]*lang.VarRootVersion
}

// Each table is immutable after publication, so call guards only need a
// pointer load when bootstrap or a test installs a replacement table.
var defaultCoreRoots atomic.Pointer[coreRootTable]

func recordDefaultCoreRoots(core *lang.Namespace, names ...string) {
	roots := make(map[*lang.Var]*lang.VarRootVersion, len(names))
	for _, name := range names {
		vr := core.FindInternedVar(lang.NewSymbol(name))
		if vr != nil && vr.IsBound() {
			roots[vr] = vr.RootVersion()
		}
	}
	defaultCoreRoots.Store(&coreRootTable{byVar: roots})
}

// IsDefaultCoreVar reports whether vr still has the standard root installed
// during runtime initialization. Generated optimizations use this guard before
// replacing higher-order core operations whose Vars remain redefinable.
func IsDefaultCoreVar(vr *lang.Var) bool {
	table := defaultCoreRoots.Load()
	if table == nil {
		return false
	}
	version := table.byVar[vr]
	return version != nil && vr.RootVersion() == version &&
		!vr.IsMacro() && !vr.IsDynamic()
}
