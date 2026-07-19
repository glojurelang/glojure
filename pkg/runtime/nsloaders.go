package runtime

import (
	"strings"
	"sync"

	"github.com/glojurelang/glojure/internal/goid"
	"github.com/glojurelang/glojure/pkg/lang"
)

type nsLoader struct {
	load func()

	mu      sync.Mutex
	cond    *sync.Cond
	loaded  bool
	loading int64
}

var (
	// nsLoaders is a map of namespace resource names to their loader
	// functions. Used for pre-compiled namespaces.
	nsLoaders = map[string]*nsLoader{}
)

func init() {
	lang.SetUnboundVarResolver(ensureVarNamespaceLoaded)
}

// RegisterNSLoader registers a loader function for a namespace given its resource name
// (i.e. root path with slashes, no extension).
func RegisterNSLoader(nsResource string, loader func()) {
	if _, exists := nsLoaders[nsResource]; exists {
		panic("namespace loader already registered for " + nsResource)
	}
	entry := &nsLoader{load: loader}
	entry.cond = sync.NewCond(&entry.mu)
	nsLoaders[nsResource] = entry
}

// GetNSLoader retrieves the loader function for a namespace given its resource name.
func GetNSLoader(nsResource string) func() {
	entry := nsLoaders[nsResource]
	if entry == nil {
		return nil
	}
	return func() {
		entry.run(false)
	}
}

func ensureVarNamespaceLoaded(v *lang.Var) {
	resource := strings.ReplaceAll(v.Namespace().Name().String(), ".", "/")
	if entry := nsLoaders[resource]; entry != nil {
		entry.run(true)
	}
}

// run invokes an AOT loader. Lazy loads run at most once, while explicit
// loads continue to support reload semantics.
func (l *nsLoader) run(lazy bool) {
	gid := goid.Get()

	l.mu.Lock()
	if lazy && l.loaded {
		l.mu.Unlock()
		return
	}
	for l.loading != 0 && l.loading != gid {
		l.cond.Wait()
		if lazy && l.loaded {
			l.mu.Unlock()
			return
		}
	}
	if l.loading == gid {
		l.mu.Unlock()
		return
	}
	l.loading = gid
	l.mu.Unlock()

	loaded := false
	defer func() {
		l.mu.Lock()
		l.loading = 0
		if loaded {
			l.loaded = true
		}
		l.cond.Broadcast()
		l.mu.Unlock()
	}()

	l.load()
	loaded = true
}
