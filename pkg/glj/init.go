package glj

import (
	"io"
	"os"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/runtime"
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
