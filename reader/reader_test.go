package reader

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/value"
	"github.com/kylelemons/godebug/diff"
)

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
		outData, err := ioutil.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		testCases = append(testCases, testCase{
			name:   filepath.Base(path),
			input:  string(data),
			output: string(outData),
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := New(strings.NewReader(tc.input))
			exprs, err := r.ReadAll()
			if err != nil {
				t.Fatal(err)
			}

			strs := make([]string, len(exprs)+1)
			strs[len(strs)-1] = ""
			for i, expr := range exprs {
				strs[i] = value.ToString(expr)
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
			value.ToString(expr)
		}
	})
}
