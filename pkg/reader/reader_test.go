package reader

import (
	"bytes"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	value "github.com/glojurelang/glojure/pkg/lang"
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
		data, err := readFile(path)
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
			outData, err = readFile(outPath)
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

	aliasNS := value.FindOrCreateNamespace(value.NewSymbol("resolved.alias"))
	ns := value.FindOrCreateNamespace(value.NewSymbol("user"))
	ns.AddAlias(value.NewSymbol("aliased"), aliasNS)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := New(strings.NewReader(tc.input),
				WithFilename(tc.name),
				// WithSymbolResolver(&testSymbolResolver{}), // TODO: option to test with this
				WithGetCurrentNS(func() *value.Namespace {
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
		data, err := readFile(path)
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
		data, err := readFile(path)
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

func testPrintString(x interface{}) string {
	value.PushThreadBindings(value.NewMap(
		value.VarPrintReadably, true,
	))
	defer value.PopThreadBindings()

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

	return value.PrintString(x)
}

func readFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return normalizeLineEndings(data), nil
}

func normalizeLineEndings(buf []byte) []byte {
	return bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n"))
}
