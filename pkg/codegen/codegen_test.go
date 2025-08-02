package codegen

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestCodegen(t *testing.T) {
	testFiles, err := filepath.Glob("testdata/*.glj")
	if err != nil {
		t.Fatal(err)
	}

	for _, testFile := range testFiles {
		testName := strings.TrimSuffix(filepath.Base(testFile), ".glj")
		t.Run(testName, func(t *testing.T) {
			// Read input file
			input, err := ioutil.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			// Parse and analyze
			nodes, err := parseAndAnalyze(string(input))
			if err != nil {
				t.Fatalf("failed to parse/analyze: %v", err)
			}

			// Generate code
			var buf bytes.Buffer
			gen := New(&buf)
			if err := gen.Generate(nodes); err != nil {
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

			// TODO: Compile and run the generated code to verify behavior
			// This will be added once we have more complete code generation
		})
	}
}

func parseAndAnalyze(input string) ([]*ast.Node, error) {
	// Create reader without needing full environment
	r := reader.New(strings.NewReader(input), reader.WithFilename("test.glj"),
		reader.WithGetCurrentNS(func() *lang.Namespace {
			// Return a minimal namespace for testing
			return lang.FindOrCreateNamespace(lang.NewSymbol("user"))
		}))

	// Create analyzer with minimal setup
	analyzer := &compiler.Analyzer{
		Macroexpand1: func(form interface{}) (interface{}, error) {
			// For now, no macro expansion in tests
			return form, nil
		},
		CreateVar: func(sym *lang.Symbol, env compiler.Env) (interface{}, error) {
			// Create a var in the current namespace
			ns := lang.FindOrCreateNamespace(lang.NewSymbol("user"))
			return ns.Intern(sym), nil
		},
		IsVar: func(v interface{}) bool {
			_, ok := v.(*lang.Var)
			return ok
		},
		Gensym: func(prefix string) *lang.Symbol {
			// Simple gensym for testing
			return lang.NewSymbol(fmt.Sprintf("%s_%d", prefix, 1))
		},
		FindNamespace: func(sym *lang.Symbol) *lang.Namespace {
			return lang.FindNamespace(sym)
		},
	}

	// Parse and analyze all forms
	var nodes []*ast.Node
	for {
		form, err := r.ReadOne()
		if err == reader.ErrEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		node, err := analyzer.Analyze(form, lang.NewMap())
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// TestBehavior verifies that generated code produces the same results as interpreted code
func TestBehavior(t *testing.T) {
	// This test will be implemented once we can compile and run generated code
	t.Skip("Behavioral testing not yet implemented")
}
