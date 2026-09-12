//go:build !glj_aot_runtime

package runtime_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

func TestPortableStdlibCodegenDiagnostics(t *testing.T) {
	if runtime.GetUseAOT() {
		t.Skip("requires GLOJURE_USE_AOT=false and glj_no_aot_stdlib")
	}
	// Reload source roots, replacing native bootstrap overrides before codegen.
	runtime.RT.Load("clojure/core")
	for _, namespace := range []string{"clojure.core", "clojure.test", "glojure.go.io"} {
		glj.Var("clojure.core", "require").Invoke(lang.NewSymbol(namespace))
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		previous := os.Stdout
		os.Stdout = write
		captured := make(chan string, 1)
		go func() {
			output, _ := io.ReadAll(read)
			captured <- string(output)
		}()
		func() {
			defer func() { os.Stdout = previous; write.Close() }()
			var output bytes.Buffer
			err = runtime.NewGenerator(&output).Generate(
				lang.FindNamespace(lang.NewSymbol(namespace)))
		}()
		output := <-captured
		read.Close()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output, "Warning:") ||
			strings.Contains(output, "generating nil") {
			t.Fatalf("%s generated unsupported code:\n%s", namespace, output)
		}
	}
}
