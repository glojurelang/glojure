//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
)

// compileNSToFile compiles the given namespace to a Go source file,
// given a fs.FS and the script base name (without extension).
func compileNSToFile(filesystem fs.FS, scriptBase string) {
	// os.DirFS is a named string type. Other filesystem implementations do not
	// provide a writable path for the generated loader.
	if reflect.TypeOf(filesystem).Kind() != reflect.String {
		panic(fmt.Errorf("cannot compile %s: filesystem is not writable", scriptBase))
	}
	fsDir := fmt.Sprintf("%s", filesystem)
	generateNamespaceAOT(fsDir, lang.VarCurrentNS.Deref().(*lang.Namespace))
}

func generateNamespaceAOT(fsDir string, ns *lang.Namespace) {
	path := nsToPath(ns.Name().Name())
	targetDir := filepath.Join(fsDir, path)
	targetFile := filepath.Join(targetDir, "loader.go")

	fmt.Printf("Compiling %s to %s\n", ns.Name(), targetFile)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	gen := newGenerator(&buf, aotDirectLinkCoreEnabled())
	if err := gen.Generate(ns); err != nil {
		_ = os.WriteFile(targetFile, buf.Bytes(), 0644)
		panic(fmt.Errorf("failed to generate code for namespace %s: %w", ns.Name(), err))
	}
	if err := os.WriteFile(targetFile, buf.Bytes(), 0644); err != nil {
		panic(fmt.Errorf("failed to write generated code to %s: %w", targetFile, err))
	}
}

func aotDirectLinkCoreEnabled() bool {
	compilerOptions := lang.NSCore.FindInternedVar(
		lang.NewSymbol("*compiler-options*"),
	)
	if compilerOptions == nil || !compilerOptions.IsBound() {
		return true
	}
	options := compilerOptions.Get()
	missing := &struct{}{}
	value := lang.GetDefault(
		options,
		lang.KWDirectLinking,
		missing,
	)
	if value == missing {
		return true
	}
	return RT.BooleanCast(value)
}
