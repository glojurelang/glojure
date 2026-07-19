//go:build glj_no_aot_stdlib

package glj

import (
	"net/url"
	"reflect"

	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/google/uuid"
)

// Register the host values used while interpreting and regenerating the
// standard library. Production builds use the generated AOT loaders and do not
// need these bootstrap-only exports in their default interop registry.
func init() {
	pkgmap.Set("net/url.URL", reflect.TypeOf((*url.URL)(nil)).Elem())
	pkgmap.Set("net/url.*URL", reflect.TypeOf((*url.URL)(nil)))
	pkgmap.Set("net/url.Parse", url.Parse)
	pkgmap.Set("net/url.ParseRequestURI", url.ParseRequestURI)

	pkgmap.Set("github.com/google/uuid.UUID", reflect.TypeOf(uuid.UUID{}))
	pkgmap.Set("github.com/google/uuid.Parse", uuid.Parse)
	pkgmap.Set("github.com/google/uuid.NewRandom", uuid.NewRandom)
	pkgmap.Set("github.com/google/uuid.NewV7", uuid.NewV7)
}
