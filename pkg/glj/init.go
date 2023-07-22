package glj

import (
	"io"
	"os"

	"github.com/glojurelang/glojure/pkg/gen/gljimports"
	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/glojurelang/glojure/pkg/runtime"
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
