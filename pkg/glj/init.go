package glj

import (
	"io"
	"os"

	// Add the Go standard library to the pkgmap.
	_ "github.com/glojurelang/glojure/pkg/gen/gljimports"
	"github.com/glojurelang/glojure/pkg/lang"

	// Add NS loaders for the standard library.
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/core"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/core/async"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/go/io"
	_ "github.com/glojurelang/glojure/pkg/stdlib/glojure/protocols"

	"github.com/glojurelang/glojure/pkg/runtime"
)

func init() {
	initEnv(os.Stdout)
}

func initEnv(stdout io.Writer) lang.Environment {
	// TODO: clean up this code. copied from rtcompat.go.
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*lang.Var{lang.VarCurrentNS, lang.VarWarnOnReflection, lang.VarUncheckedMath, lang.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	lang.PushThreadBindings(lang.NewMap(kvs...))

	return runtime.NewEnvironment(runtime.WithStdout(stdout))
}
