package runtime

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/value"
	"github.com/kylelemons/godebug/diff"
)

func TestEval(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		output string
	}

	var testCases = []testCase{}

	// read all *.in files in testdata/eval as test cases.
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

			// save stdout to a buffer
			stdout := &strings.Builder{}

			rdr := reader.New(strings.NewReader(tc.input), reader.WithFilename(tc.name))
			forms, err := rdr.ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			env := NewEnvironment(WithStdout(stdout), WithLoadPath([]string{"testdata/eval"}))
			_, err = env.Eval(value.NewList([]interface{}{
				value.NewSymbol("ns"),
				SymbolUserNamespace,
			}))
			if err != nil {
				t.Fatal(err)
			}

			for _, form := range forms {
				_, err := env.Eval(form)
				if err != nil {
					t.Fatal(err)
				}
			}

			if got, want := stdout.String(), tc.output; got != want {
				t.Errorf("diff (-want,+got):\n%s", diff.Diff(want, got))
			}
		})
	}
}

func TestEvalErrors(t *testing.T) {
	type testCase struct {
		name    string
		input   string
		errorS  string
		errorRE *regexp.Regexp
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
		if !strings.HasPrefix(lines[0], ";;;ERROR=") && !strings.HasPrefix(lines[0], ";;;ERROR_RE=") {
			t.Fatalf("no error line in %s", path)
		}
		errorS := strings.Replace(strings.TrimPrefix(lines[0], ";;;ERROR="), "\\n", "\n", -1)
		var errorRE *regexp.Regexp
		if strings.HasPrefix(lines[0], ";;;ERROR_RE=") {
			errorRE = regexp.MustCompile(strings.TrimPrefix(lines[0], ";;;ERROR_RE="))
		}

		testCases = append(testCases, testCase{
			name:    filepath.Base(path),
			input:   string(data),
			errorS:  errorS,
			errorRE: errorRE,
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rdr := reader.New(strings.NewReader(tc.input), reader.WithFilename(tc.name))
			forms, err := rdr.ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			env := NewEnvironment(WithStdout(io.Discard), WithLoadPath([]string{"testdata/eval_error"}))
			_, err = env.Eval(value.NewList([]interface{}{
				value.NewSymbol("ns"),
				SymbolUserNamespace,
			}))
			if err != nil {
				t.Fatal(err)
			}

			err = nil
			for _, form := range forms {
				_, err = env.Eval(form)
				if err != nil {
					break
				}
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			got := err.Error()
			if tc.errorRE != nil {
				if !tc.errorRE.MatchString(got) {
					t.Errorf("error does not match %q\n\n%s", tc.errorRE, got)
				}
			} else {
				if want := tc.errorS; got != want {
					t.Errorf("diff (-want,+got):\n%s", diff.Diff(want, got))
				}
			}
		})
	}
}
