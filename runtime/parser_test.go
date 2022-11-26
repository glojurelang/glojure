package runtime

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/diff"
)

func TestParse(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		output string
	}

	var testCases = []testCase{}

	// read all *.in files in testdata/parser as test cases.
	paths, err := filepath.Glob("testdata/eval/*.glj")
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

			prog, err := Parse(strings.NewReader(tc.input), WithFilename(tc.name))
			if err != nil {
				t.Fatal(err)
			}

			// save stdout to a buffer
			stdout := &strings.Builder{}

			_, err = prog.Eval(WithStdout(stdout), WithLoadPath([]string{"testdata/eval"}))
			if err != nil {
				t.Fatal(err)
			}

			if got, want := stdout.String(), tc.output; got != want {
				t.Errorf("\n=== got ====\n%s============\n=== want ===\n%s============", got, want)
				t.Errorf("diff (-want,+got):\n%s", diff.Diff(want, got))
			}
		})
	}
}

func TestEvalErrors(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		errorS string
	}

	var testCases = []testCase{}

	// read all *.in files in testdata/eval_error as test cases.
	paths, err := filepath.Glob("testdata/eval_error/*.in")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// expected error should be on the first line of the input, after
		// ";;;ERROR="
		lines := strings.Split(string(data), "\n")
		if len(lines) < 1 {
			t.Fatalf("no error line in %s", path)
		}
		if !strings.HasPrefix(lines[0], ";;;ERROR=") {
			t.Fatalf("no error line in %s", path)
		}
		errorS := strings.Replace(strings.TrimPrefix(lines[0], ";;;ERROR="), "\\n", "\n", -1)

		testCases = append(testCases, testCase{
			name:   filepath.Base(path),
			input:  string(data),
			errorS: errorS,
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prog, err := Parse(strings.NewReader(tc.input), WithFilename(tc.name))
			if err != nil {
				t.Fatal(err)
			}

			_, err = prog.Eval(WithStdout(io.Discard), WithLoadPath([]string{"testdata/eval_error"}))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got, want := err.Error(), tc.errorS; got != want {
				t.Errorf("\n=== got ====\n%s============\n=== want ===\n%s============", got, want)
			}
		})
	}
}
