package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"sync"
)

var coreNativeLinks struct {
	sync.Mutex
	refresh func()
	done    bool
}

// RegisterCoreNativeLinks is emitted by the core AOT loader. Bootstrap
// refreshes those direct calls once after installing context-free native
// roots. Later user redefinitions retain the normal direct-linking policy.
func RegisterCoreNativeLinks(refresh func()) {
	coreNativeLinks.Lock()
	defer coreNativeLinks.Unlock()
	coreNativeLinks.refresh, coreNativeLinks.done = refresh, false
}

func refreshCoreNativeLinks() {
	coreNativeLinks.Lock()
	defer coreNativeLinks.Unlock()
	if coreNativeLinks.refresh != nil && !coreNativeLinks.done {
		coreNativeLinks.refresh()
		coreNativeLinks.done = true
	}
}

// CoreArityFn adapts a native IFn to the dispatch type of a compiled core
// function. Existing multi-arity roots already carry the exact arity table.
func CoreArityFn(fn lang.IFn) lang.ArityFn {
	if arity, ok := fn.(lang.ArityFn); ok {
		return arity
	}
	return lang.NewArityFnMethods(map[int]lang.IFn{
		0: lang.FnFunc0(func() any { return lang.Apply0(fn) }),
		1: lang.FnFunc1(func(a any) any { return lang.Apply1(fn, a) }),
		2: lang.FnFunc2(func(a, b any) any { return lang.Apply2(fn, a, b) }),
		3: lang.FnFunc3(func(a, b, c any) any { return lang.Apply3(fn, a, b, c) }),
		4: lang.FnFunc4(func(a, b, c, d any) any { return lang.Apply4(fn, a, b, c, d) }),
		5: lang.FnFunc5(func(a, b, c, d, e any) any { return lang.Apply5(fn, a, b, c, d, e) }),
	}, fn, 0)
}
