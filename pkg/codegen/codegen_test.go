package codegen_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/codegen"
	"github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"
)

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

			generateAndTestNamespace(t, ns, strings.TrimSuffix(testFile, ".glj")+".go")
		})
	}

	t.Run("glojure.core", func(t *testing.T) {
		// Test the core namespace
		ns := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		if ns == nil {
			t.Fatal("glojure.core namespace not found")
		}

		goldenFile := "testdata/codegen/test/core.go"
		generateAndTestNamespace(t, ns, goldenFile)
	})
}

func generateAndTestNamespace(t *testing.T, ns *lang.Namespace, goldenFile string) {
	t.Helper()

	// Generate code for the namespace
	var buf bytes.Buffer
	gen := codegen.New(&buf)
	if err := gen.Generate(ns); err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	generated := buf.Bytes()

	updateGolden := os.Getenv("UPDATE_SNAPSHOT") == "1"
	if updateGolden {
		if err := ioutil.WriteFile(goldenFile, generated, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Compare with golden file
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

	// Check if namespace has -main function with expected output
	// TODO: consider dropping this; we really just want to ensure
	// the interpreter, here, behaves the same as the generated code
	testMainFunction(t, ns)
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

// testMainFunction tests the -main function if it exists and has :expected-output or :expected-error metadata
func testMainFunction(t *testing.T, ns *lang.Namespace) {
	// Look for -main var in the namespace
	mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
	if mainVar == nil {
		// No -main function, nothing to test
		return
	}

	// Check if -main has :expected-output or :expected-error metadata
	meta := mainVar.Meta()
	if meta == nil {
		return
	}

	expectedOutput := meta.ValAt(lang.NewKeyword("expected-output"))
	expectedError := meta.ValAt(lang.NewKeyword("expected-error"))

	if expectedOutput == nil && expectedError == nil {
		return
	}

	// If we expect an error, use recover to catch it
	if expectedError != nil {
		defer func() {
			if r := recover(); r != nil {
				// Check if the panic matches expected error
				if !lang.Equals(r, expectedError) {
					t.Errorf("-main panicked with %v, expected %v", r, expectedError)
				}
			} else {
				t.Errorf("-main should have panicked with %v, but didn't", expectedError)
			}
		}()

		// Run -main - should panic
		mainVar.Invoke()
		return
	}

	// Run -main and check the result
	result := mainVar.Invoke()
	if !lang.Equals(result, expectedOutput) {
		t.Errorf("-main returned %v, expected %v", result, expectedOutput)
	}
}
