package runtime

import (
	"fmt"
	"io"
	"net/url"
	"sync/atomic"
)

// URLOpener opens a non-file URL for reading.
type URLOpener func(*url.URL) (io.ReadCloser, error)

var registeredURLOpener atomic.Value

// RegisterURLOpener installs the process-wide implementation used by
// glojure.go.io for non-file URLs. The most recent registration wins.
func RegisterURLOpener(opener URLOpener) {
	if opener == nil {
		panic("runtime: cannot register a nil URL opener")
	}
	registeredURLOpener.Store(opener)
}

// OpenURL opens a non-file URL using the registered implementation.
func OpenURL(u *url.URL) (io.ReadCloser, error) {
	opener, ok := registeredURLOpener.Load().(URLOpener)
	if !ok {
		return nil, fmt.Errorf("no URL opener is linked for %q", u)
	}
	return opener(u)
}
