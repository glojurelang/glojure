package codegengotest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/glojurelang/glojure/pkg/lang"
)

// TestMain is the entry point for running tests. We use it to ensure
// that this test runs in a separate process.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestGeneratedGo(t *testing.T) {
	// Find all .out files in testdata directory
	testdataDir := "../testdata"
	var outFiles []string

	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "load.go.out") {
			outFiles = append(outFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Error walking testdata: %v", err)
	}

	for _, outFile := range outFiles {
		outFile := outFile // Capture range variable
		// Extract package name from path (e.g., "const_keyword" from "codegen/test/const_keyword/load.go.out")
		dir := filepath.Dir(outFile)
		pkgName := filepath.Base(dir)

		t.Run(pkgName, func(t *testing.T) {
			t.Parallel()

			// Read the corresponding .glj file to check for -main metadata
			gljFile := strings.TrimSuffix(outFile, "/load.go.out") + ".glj"
			nsName, hasMain := getNamespaceMetadata(t, gljFile)

			if !hasMain {
				t.Skip("No -main function with expected metadata")
			}

			// Create temp directory for the Go program
			tempDir, err := ioutil.TempDir("", "codegengo_test_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create package directory in temp dir
			pkgDir := filepath.Join(tempDir, pkgName)
			if err := os.MkdirAll(pkgDir, 0755); err != nil {
				t.Fatalf("Failed to create package dir: %v", err)
			}

			// Copy .out file to package dir as .go file
			outContent, err := ioutil.ReadFile(outFile)
			if err != nil {
				t.Fatalf("Failed to read .out file: %v", err)
			}

			goFile := filepath.Join(pkgDir, "load.go")
			if err := ioutil.WriteFile(goFile, outContent, 0644); err != nil {
				t.Fatalf("Failed to write .go file: %v", err)
			}

			// Generate main.go
			mainContent := generateMainFile(pkgName, nsName)
			mainFile := filepath.Join(tempDir, "main.go")
			if err := ioutil.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
				t.Fatalf("Failed to write main.go: %v", err)
			}

			// Get absolute path to project root (we're in pkg/codegen/codegengotest)
			projectRoot, err := filepath.Abs("../../..")
			if err != nil {
				t.Fatalf("Failed to get project root: %v", err)
			}

			// Create go.mod with absolute path replacement
			goModContent := fmt.Sprintf(`module testprog

go 1.21

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, projectRoot)
			goModFile := filepath.Join(tempDir, "go.mod")
			if err := ioutil.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			// Copy go.sum from project root
			goSumSrc := filepath.Join(projectRoot, "go.sum")
			goSumContent, err := ioutil.ReadFile(goSumSrc)
			if err != nil {
				t.Fatalf("Failed to read go.sum: %v", err)
			}
			goSumDst := filepath.Join(tempDir, "go.sum")
			if err := ioutil.WriteFile(goSumDst, goSumContent, 0644); err != nil {
				t.Fatalf("Failed to write go.sum: %v", err)
			}

			// Run go mod tidy to ensure all dependencies are resolved
			cmd := exec.Command("go", "mod", "tidy")
			cmd.Dir = tempDir
			var tidyStderr bytes.Buffer
			cmd.Stderr = &tidyStderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to run go mod tidy: %v\nStderr: %s", err, tidyStderr.String())
			}

			// Build the program
			cmd = exec.Command("go", "build", "-o", "testprog", ".")
			cmd.Dir = tempDir
			var buildStderr bytes.Buffer
			cmd.Stderr = &buildStderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to build program: %v\nStderr: %s", err, buildStderr.String())
			}

			// Run the program
			cmd = exec.Command("./testprog")
			cmd.Dir = tempDir
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = cmd.Run()

			if err != nil {
				// Program exited with non-zero status - test failed
				t.Errorf("Test failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
			}
		})
	}
}

// getNamespaceMetadata reads a .glj file and extracts namespace and checks for -main
func getNamespaceMetadata(t *testing.T, gljFile string) (nsName string, hasMain bool) {
	t.Helper()

	content, err := ioutil.ReadFile(gljFile)
	if err != nil {
		t.Logf("Could not read .glj file %s: %v", gljFile, err)
		return "", false
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "(ns ") {
			// Extract namespace name
			nsLine := strings.TrimPrefix(line, "(ns ")
			nsLine = strings.TrimSuffix(nsLine, ")")
			parts := strings.Fields(nsLine)
			if len(parts) > 0 {
				nsName = parts[0]
			}
		}
		// Look for -main definition with metadata
		// Check if current line has metadata with :expected- or next line has defn -main
		if strings.Contains(line, ":expected-") {
			// Check if this line or next line has -main
			if strings.Contains(line, "-main") {
				hasMain = true
			} else if i+1 < len(lines) && strings.Contains(lines[i+1], "-main") {
				hasMain = true
			}
		}
	}

	// If we couldn't find it from the file, try loading the namespace to check
	if nsName != "" && !hasMain {
		ns := lang.FindNamespace(lang.NewSymbol(nsName))
		if ns != nil {
			mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
			if mainVar != nil {
				meta := mainVar.Meta()
				if meta != nil {
					if out := meta.ValAt(lang.NewKeyword("expected-output")); out != nil {
						hasMain = true
					}
					if throw := meta.ValAt(lang.NewKeyword("expected-throw")); throw != nil {
						hasMain = true
					}
				}
			}
		}
	}

	return nsName, hasMain
}

var mainTemplate = template.Must(template.New("main").Parse(`package main

import (
	"fmt"
	"os"

	testpkg "testprog/{{.PkgName}}"
	"github.com/glojurelang/glojure/pkg/lang"
  _ "github.com/glojurelang/glojure/pkg/glj"
)

func main() {
	testpkg.LoadNS()

	ns := lang.FindNamespace(lang.NewSymbol("{{.NsName}}"))
	if ns == nil {
		fmt.Println("ERROR: namespace not found")
		os.Exit(1)
	}

	mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
	if mainVar == nil {
		fmt.Println("ERROR: -main function not found")
		os.Exit(1)
	}

	// Get metadata for expected output/throw
	meta := mainVar.Meta()
	if meta == nil {
		fmt.Println("ERROR: -main has no metadata")
		os.Exit(1)
	}

	expectedOutput := meta.ValAt(lang.NewKeyword("expected-output"))
	expectedThrow := meta.ValAt(lang.NewKeyword("expected-throw"))

	if expectedOutput == nil && expectedThrow == nil {
		fmt.Println("ERROR: -main has no :expected-output or :expected-throw metadata")
		os.Exit(1)
	}

	if expectedThrow != nil {
		// Expect a panic/throw
		defer func() {
			if r := recover(); r != nil {
				if lang.Equals(r, expectedThrow) {
					fmt.Println("SUCCESS: Got expected throw")
					os.Exit(0)
				} else {
					fmt.Printf("FAIL: Expected throw %v, got %v\n", expectedThrow, r)
					os.Exit(1)
				}
			} else {
				fmt.Printf("FAIL: Expected throw %v, but no panic occurred\n", expectedThrow)
				os.Exit(1)
			}
		}()

		// Run -main - should panic
		mainVar.Invoke()
		fmt.Printf("FAIL: Expected throw %v, but no panic occurred\n", expectedThrow)
		os.Exit(1)
	} else {
		// Expect normal return value
		result := mainVar.Invoke()
		if lang.Equals(result, expectedOutput) {
			fmt.Println("SUCCESS: Got expected output")
			os.Exit(0)
		} else {
			fmt.Printf("FAIL: Expected %v, got %v\n", expectedOutput, result)
			os.Exit(1)
		}
	}
}
`))

// generateMainFile generates a main.go file that imports and calls LoadNS
func generateMainFile(pkgName, nsName string) string {
	var buf bytes.Buffer
	err := mainTemplate.Execute(&buf, struct {
		PkgName string
		NsName  string
	}{
		PkgName: pkgName,
		NsName:  nsName,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to generate main.go: %v", err))
	}
	return buf.String()
}
