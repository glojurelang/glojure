package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// DirectSwap0 is the uncommon fallback for compiled non-escaping swap!
// callbacks whose target is an IAtom implementation other than *lang.Atom.
func DirectSwap0(target any, callback lang.IFn) any {
	if fixed, ok := target.(fixedAtomSwap0); ok {
		return fixed.Swap0(callback)
	}
	if atom, ok := target.(lang.IAtom); ok {
		return atom.Swap(callback, nil)
	}
	panic(lang.NewIllegalArgumentError("swap! target does not implement IAtom"))
}
