package glj

import (
	"io"
	"os"

	"github.com/glojurelang/glojure/gen/gljimports"
	"github.com/glojurelang/glojure/pkgmap"
	"github.com/glojurelang/glojure/runtime"
	value "github.com/glojurelang/glojure/pkg/lang"
)

func init() {
	initEnv(os.Stdout)
}

func initEnv(stdout io.Writer) value.Environment {
	gljimports.RegisterImports(pkgmap.Set)

	// TODO: clean up this code. copied from rtcompat.go.
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*value.Var{value.VarCurrentNS, value.VarWarnOnReflection, value.VarUncheckedMath, value.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	value.PushThreadBindings(value.NewMap(kvs...))

	return runtime.NewEnvironment(runtime.WithStdout(stdout))
}
