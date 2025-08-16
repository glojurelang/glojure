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

const testHarnessCode = `package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	
	_ "github.com/glojurelang/glojure/pkg/codegen/testdata/codegen/test"
	"github.com/glojurelang/glojure/pkg/lang"
)

func main() {
	// Find all .glj files in testdata directory
	// Get the testdata path relative to GOPATH or module root
	testdataDir := os.Args[1]
	var namespaces []string
	
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".glj") {
			// Read first line to get namespace
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "(ns ") {
				// Extract namespace name
				nsLine := lines[0]
				nsLine = strings.TrimPrefix(nsLine, "(ns ")
				nsLine = strings.TrimSuffix(nsLine, ")")
				parts := strings.Fields(nsLine)
				if len(parts) > 0 {
					namespaces = append(namespaces, parts[0])
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking testdata: %v\n", err)
		os.Exit(1)
	}

	failed := false
	for _, nsName := range namespaces {
		ns := lang.FindNamespace(lang.NewSymbol(nsName))
		if ns == nil {
			fmt.Printf("SKIP: namespace %s not found\n", nsName)
			continue
		}

		mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
		if mainVar == nil {
			fmt.Printf("SKIP: %s/-main not found\n", nsName)
			continue
		}

		// Check if -main has :expected-output metadata
		meta := mainVar.Meta()
		if meta == nil {
			fmt.Printf("SKIP: %s/-main has no metadata\n", nsName)
			continue
		}

		expected := meta.ValAt(lang.NewKeyword("expected-output"))
		if expected == nil {
			fmt.Printf("SKIP: %s/-main has no :expected-output\n", nsName)
			continue
		}

		// Run -main and check the result
		result := mainVar.Invoke()
		if !lang.Equals(result, expected) {
			fmt.Printf("FAIL: %s/-main returned %v, expected %v\n", nsName, result, expected)
			failed = true
		} else {
			fmt.Printf("PASS: %s/-main\n", nsName)
		}
	}

	if failed {
		os.Exit(1)
	}
}
`

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

			// Check if namespace has -main function with expected output
			testMainFunction(t, ns)
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

// testMainFunction tests the -main function if it exists and has :expected-output metadata
func testMainFunction(t *testing.T, ns *lang.Namespace) {
	// Look for -main var in the namespace
	mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
	if mainVar == nil {
		// No -main function, nothing to test
		return
	}

	// Check if -main has :expected-output metadata
	meta := mainVar.Meta()
	if meta == nil {
		return
	}

	expectedOutput := meta.ValAt(lang.NewKeyword("expected-output"))
	if expectedOutput == nil {
		return
	}

	// Run -main and check the result
	result := mainVar.Invoke()
	if !lang.Equals(result, expectedOutput) {
		t.Errorf("-main returned %v, expected %v", result, expectedOutput)
	}
}

// TestGeneratedCode compiles and runs the generated code to verify behavior
func TestGeneratedCode(t *testing.T) {
	// Write the test harness in a temporary directory
	tmpDir, err := ioutil.TempDir("", "glojure_test_harness")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	harnessPath := filepath.Join(tmpDir, "harness_main.go")
	if err := ioutil.WriteFile(harnessPath, []byte(testHarnessCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Get absolute path to testdata directory
	testdataPath, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	// Compile and run the test harness
	cmd := exec.Command("go", "run", harnessPath, testdataPath)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Test harness failed: %v\nOutput:\n%s", err, output)
	}

	// Print the output for visibility
	t.Logf("Test harness output:\n%s", output)
}
