package codegen

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestCodegen(t *testing.T) {
	var testFiles []string
	err := filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".glj") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Sort test files for consistent ordering
	sort.Strings(testFiles)

	for i, testFile := range testFiles {
		baseName := strings.TrimSuffix(filepath.Base(testFile), ".glj")
		testName := fmt.Sprintf("%02d_%s", i+1, baseName)
		t.Run(testName, func(t *testing.T) {
			// Parse test file to get namespace name
			nsName := getNamespaceFromFile(t, testFile)
			if nsName == "" {
				// If no namespace declaration, use the filename as namespace
				nsName = strings.TrimSuffix(filepath.Base(testFile), ".glj")
				nsName = strings.ReplaceAll(nsName, "_", "-")
				nsName = strings.ReplaceAll(nsName, ".", "-")
			}

			require := glj.Var("glojure.core", "require")
			runtime.AddLoadPath(os.DirFS("testdata"))
			// Load the namespace
			require.Invoke(lang.NewSymbol(nsName))

			ns := lang.FindNamespace(lang.NewSymbol(nsName))

			// Generate code for the namespace
			var buf bytes.Buffer
			gen := New(&buf)
			if err := gen.Generate(ns); err != nil {
				t.Fatalf("failed to generate code: %v", err)
			}

			generated := buf.Bytes()

			// Compare with golden file
			goldenFile := strings.TrimSuffix(testFile, ".glj") + ".go"
			if *updateGolden {
				if err := ioutil.WriteFile(goldenFile, generated, 0644); err != nil {
					t.Fatal(err)
				}
			}

			expected, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}

			if !bytes.Equal(generated, expected) {
				t.Errorf("generated code does not match golden file.\nGenerated:\n%s\nExpected:\n%s",
					generated, expected)
			}

			// run go vet on the output file. print any errors from stderr
			cmd := exec.Command("go", "vet", "-all", goldenFile)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Errorf("go vet failed for %s: %v\nStderr:\n%s", goldenFile, err, stderr.String())
			}

			// TODO: Compile and run the generated code to verify behavior
			// This will be added once we have more complete code generation
		})
	}
}

// getNamespaceFromFile attempts to extract the namespace declaration from a file
func getNamespaceFromFile(t *testing.T, filename string) string {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}

	r := reader.New(strings.NewReader(string(input)),
		reader.WithFilename(filename),
	)

	// Look for first form, check if it's an ns declaration
	form, err := r.ReadOne()
	if err != nil {
		return ""
	}

	// Check if it's a list starting with 'ns
	if list, ok := form.(lang.ISeq); ok {
		first := lang.First(list)
		if sym, ok := first.(*lang.Symbol); ok && sym.Name() == "ns" {
			// Get the namespace name (second element)
			second := lang.First(lang.Next(list))
			if nsSym, ok := second.(*lang.Symbol); ok {
				return nsSym.Name()
			}
		}
	}

	panic("expected namespace declaration in " + filename)
}

// TestBehavior verifies that generated code produces the same results as interpreted code
func TestBehavior(t *testing.T) {
	// This test will be implemented once we can compile and run generated code
	t.Skip("Behavioral testing not yet implemented")
}
