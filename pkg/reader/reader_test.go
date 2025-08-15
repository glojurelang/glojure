package reader

import (
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/kylelemons/godebug/diff"
)

// set GLOJ_READER_TEST_WRITE_OUTPUT=1 to write the output of the
// reader as the gold output on a failure. This is useful to
// initialize the output for a new test case.
func TestRead(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		output string
	}

	var testCases = []testCase{}

	// read all *.glj files in testdata/reader as test cases.
	paths, err := filepath.Glob("testdata/reader/*.glj")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// read corresponding *.out file.
		outPath := strings.TrimSuffix(path, ".glj") + ".out"
		var outData []byte
		// if no *.out file exists, use the input as output.
		if _, err := os.Stat(outPath); os.IsNotExist(err) {
			outData = data
		} else {
			outData, err = ioutil.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
		}
		testCases = append(testCases, testCase{
			name:   filepath.Base(path),
			input:  string(data),
			output: string(outData),
		})
	}

	aliasNS := lang.FindOrCreateNamespace(lang.NewSymbol("resolved.alias"))
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("user"))
	ns.AddAlias(lang.NewSymbol("aliased"), aliasNS)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := New(strings.NewReader(tc.input),
				WithFilename(tc.name),
				// WithSymbolResolver(&testSymbolResolver{}), // TODO: option to test with this
				WithGetCurrentNS(func() *lang.Namespace {
					return ns
				}),
			)
			exprs, err := r.ReadAll()
			if err != nil {
				t.Fatal(err)
			}

			strs := make([]string, len(exprs)+1)
			strs[len(strs)-1] = ""
			for i, expr := range exprs {
				strs[i] = testPrintString(expr)
			}
			output := strings.Join(strs, "\n")
			if output != tc.output {
				t.Errorf("diff (-want,+got):\n%s", diff.Diff(tc.output, output))
				if os.Getenv("GLOJ_READER_TEST_WRITE_OUTPUT") != "" {
					if err := os.WriteFile("testdata/reader/"+strings.TrimSuffix(tc.name, ".glj")+".out", []byte(output), 0644); err != nil {
						panic(err)
					}
				}
			}
		})
	}
}

func TestReadErrors(t *testing.T) {
	type testCase struct {
		name      string
		input     string
		outputErr string
	}

	var testCases = []testCase{}

	// read all *.in files in testdata/reader_error as test cases.
	paths, err := filepath.Glob("testdata/reader_error/*.glj")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		inString := string(data)
		if !strings.HasPrefix(inString, ";;;ERROR: ") {
			t.Fatalf("test case %s does not contain expected error string", path)
		}
		errString := strings.Split(strings.TrimPrefix(inString, ";;;ERROR: "), "\n")[0]
		testCases = append(testCases, testCase{
			name:      filepath.Base(path),
			input:     inString,
			outputErr: errString,
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := New(strings.NewReader(tc.input))
			res, err := r.ReadAll()
			if err == nil {
				t.Errorf("expected error, got nil")
				t.Fatalf("output:\n%s", res)
			}
			if err.Error() != tc.outputErr {
				t.Errorf("error mismatch:\nwant:\n%s\nhave:\n%s\n", tc.outputErr, err.Error())
			}
		})
	}
}

func FuzzRead(f *testing.F) {
	paths, err := filepath.Glob("testdata/reader/*.glj")
	if err != nil {
		f.Fatal(err)
	}
	paths2, err := filepath.Glob("testdata/reader_error/*.glj")
	if err != nil {
		f.Fatal(err)
	}
	paths = append(paths, paths2...)
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(string(data))
	}

	f.Fuzz(func(t *testing.T, program string) {
		r := New(strings.NewReader(program))
		// ignore result, we're only interested in whether it panics or hangs.
		exprs, err := r.ReadAll()
		if err != nil {
			return
		}
		for _, expr := range exprs {
			testPrintString(expr)
		}
	})
}

func TestDiscardMacro(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic discard",
			input:    "#_(prn \"discarded\") (prn \"kept\")",
			expected: []string{"(prn \"kept\")"},
		},
		{
			name:     "multiple discards",
			input:    "#_(prn \"first discarded\") #_(prn \"second discarded\") (prn \"kept\")",
			expected: []string{"(prn \"kept\")"},
		},
		{
			name:     "discard at end",
			input:    "(prn \"first\") (prn \"second\") #_(prn \"last discarded\")",
			expected: []string{"(prn \"first\")", "(prn \"second\")"},
		},
		{
			name:     "discard with nested forms",
			input:    "#_(defn ignored [] (prn \"ignored\")) (defn kept [] (prn \"kept\"))",
			expected: []string{"(defn kept [] (prn \"kept\"))"},
		},
		{
			name:     "discard with complex structures",
			input:    "#_(def ignored-map {:a 1 :b 2 :c 3}) (def kept-vector [1 2 3])",
			expected: []string{"(def kept-vector [1 2 3])"},
		},
		{
			name:     "discard with metadata",
			input:    "#_^{:tag String} (prn \"ignored\") ^{:tag Number} (prn \"kept\")",
			expected: []string{"^{:tag Number} (prn \"kept\")"},
		},
		{
			name:     "single discard at end",
			input:    "#_(prn \"only form discarded\")",
			expected: []string{},
		},
		{
			name:     "discard followed by whitespace only",
			input:    "#_(prn \"discarded\")  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(strings.NewReader(tt.input))
			exprs, err := r.ReadAll()
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			if len(exprs) != len(tt.expected) {
				t.Errorf("ReadAll() returned %d expressions, want %d", len(exprs), len(tt.expected))
			}

			for i, expr := range exprs {
				got := testPrintString(expr)
				if i < len(tt.expected) && got != tt.expected[i] {
					t.Errorf("expression %d = %q, want %q", i, got, tt.expected[i])
				}
			}
		})
	}
}

func TestDiscardMacroEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "discard with incomplete form",
			input:       "#_(prn",
			shouldError: true,
			errorMsg:    "unexpected end of input",
		},
		{
			name:        "discard with malformed nested form",
			input:       "#_(defn broken [",
			shouldError: true,
			errorMsg:    "unexpected end of input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(strings.NewReader(tt.input))
			_, err := r.ReadAll()

			if tt.shouldError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error message %q does not contain expected %q", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func testPrintString(x interface{}) string {
	lang.PushThreadBindings(lang.NewMap(
		lang.VarPrintReadably, true,
	))
	defer lang.PopThreadBindings()

	if v, ok := x.(float64); ok {
		if math.IsNaN(v) {
			return "##NaN"
		}
		if math.IsInf(v, 1) {
			return "##Inf"
		}
		if math.IsInf(v, -1) {
			return "##-Inf"
		}
	}

	return lang.PrintString(x)
}
