//go:build glj_no_aot_stdlib

package glj

import (
	"go/ast"
	"go/parser"
	"net/url"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/glojurelang/glojure/pkg/runtime"
	"github.com/google/uuid"
)

// Register the host values used while interpreting and regenerating the
// standard library. Production builds use the generated AOT loaders and do not
// need these bootstrap-only exports in their default interop registry.
func init() {
	pkgmap.Set("clojure.lang.ChunkBuffer", reflect.TypeOf((*lang.ChunkBuffer)(nil)))
	pkgmap.Set("clojure.lang.IChunk", reflect.TypeOf((*lang.IChunk)(nil)).Elem())
	pkgmap.Set("github.com/glojurelang/glojure/pkg/lang.*TaggedLiteral", reflect.TypeOf((*lang.TaggedLiteral)(nil)))

	pkgmap.Set("go/ast.*Ident", reflect.TypeOf((*ast.Ident)(nil)))
	pkgmap.Set("go/ast.*ArrayType", reflect.TypeOf((*ast.ArrayType)(nil)))
	pkgmap.Set("go/ast.*MapType", reflect.TypeOf((*ast.MapType)(nil)))
	pkgmap.Set("go/ast.*ChanType", reflect.TypeOf((*ast.ChanType)(nil)))
	pkgmap.Set("go/ast.*FuncType", reflect.TypeOf((*ast.FuncType)(nil)))
	pkgmap.Set("go/ast.*Ellipsis", reflect.TypeOf((*ast.Ellipsis)(nil)))
	pkgmap.Set("go/ast.*StructType", reflect.TypeOf((*ast.StructType)(nil)))
	pkgmap.Set("go/ast.SEND", ast.SEND)
	pkgmap.Set("go/ast.RECV", ast.RECV)
	pkgmap.Set("go/parser.ParseExpr", parser.ParseExpr)

	pkgmap.Set("net/url.URL", reflect.TypeOf((*url.URL)(nil)).Elem())
	pkgmap.Set("net/url.*URL", reflect.TypeOf((*url.URL)(nil)))
	pkgmap.Set("net/url.Parse", url.Parse)
	pkgmap.Set("net/url.ParseRequestURI", url.ParseRequestURI)
	pkgmap.Set("github.com/glojurelang/glojure/pkg/runtime.OpenURL", runtime.OpenURL)

	pkgmap.Set("github.com/google/uuid.UUID", reflect.TypeOf(uuid.UUID{}))
	pkgmap.Set("github.com/google/uuid.Parse", uuid.Parse)
	pkgmap.Set("github.com/google/uuid.NewRandom", uuid.NewRandom)
	pkgmap.Set("github.com/google/uuid.NewV7", uuid.NewV7)
}
