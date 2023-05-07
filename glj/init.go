package glj

import (
	"io"
	"os"

	"github.com/glojurelang/glojure/runtime"
	"github.com/glojurelang/glojure/value"
)

func init() {
	initEnv(os.Stdout)
}

func initEnv(stdout io.Writer) value.Environment {
	env := runtime.NewEnvironment(runtime.WithStdout(stdout))
	// TODO: clean up this code. copied from rtcompat.go.
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*value.Var{value.VarCurrentNS, value.VarWarnOnReflection, value.VarUncheckedMath} {
		kvs = append(kvs, vr, vr.Deref())
	}
	value.PushThreadBindings(value.NewMap(kvs...))

	return env
}
