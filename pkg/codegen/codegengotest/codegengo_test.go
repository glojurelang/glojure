package codegentest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/glojurelang/glojure/pkg/codegen/testdata/codegen/test"
	"github.com/glojurelang/glojure/pkg/lang"
)

// TestMain is the entry point for running tests. We use it to ensure
// that this test runs in a separate process.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestGeneratedGo(t *testing.T) {
	// Find all .glj files in testdata directory
	testdataDir := "../testdata"
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
		t.Fatalf("Error walking testdata: %v", err)
	}

	for _, nsName := range namespaces {
		nsName := nsName // Capture range variable
		t.Run(nsName, func(t *testing.T) {
			ns := lang.FindNamespace(lang.NewSymbol(nsName))
			if ns == nil {
				t.Fatalf("namespace %s not found", nsName)
			}

			mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
			if mainVar == nil {
				t.Skip()
			}

			// Check if -main has :expected-output metadata
			meta := mainVar.Meta()
			if meta == nil {
				t.Fatalf("metadata for %s/-main is nil", nsName)
			}

			expected := meta.ValAt(lang.NewKeyword("expected-output"))
			if expected == nil {
				t.Fatalf("no :expected-output metadata for %s/-main", nsName)
			}

			// Run -main and check the result
			result := mainVar.Invoke()
			if !lang.Equals(result, expected) {
				t.Errorf("%s/-main returned %v, expected %v", nsName, result, expected)
			}
		})
	}
}
