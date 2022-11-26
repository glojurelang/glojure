package reader

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/value"
)

func TestRead(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		output string
	}

	var testCases = []testCase{}

	// read all *.mrat files in testdata/reader as test cases.
	paths, err := filepath.Glob("testdata/reader/*.mrat")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// read corresponding *.out file.
		outPath := strings.TrimSuffix(path, ".mrat") + ".out"
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
				strs[i] = expr.String()
			}
			output := strings.Join(strs, "\n")
			if output != tc.output {
				t.Errorf("output mismatch:\nwant:\n%s\nhave:\n%s[END]\n", tc.output, output)
			}

			if strings.HasPrefix(tc.input, ";;;SKIP_PRINT_TEST") {
				return
			}

			var runeArr runeArray2D
			for _, expr := range exprs {
				printExprAtPosition(&runeArr, expr)
			}
			if runeArr.String() != tc.input {
				t.Errorf("input mismatch:\nwant:\n%s\nhave:\n%s[END]\n", tc.input, runeArr.String())
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
	paths, err := filepath.Glob("testdata/reader_error/*.mrat")
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
			_, err := r.ReadAll()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tc.outputErr {
				t.Errorf("error mismatch:\nwant:\n%s\nhave:\n%s\n", tc.outputErr, err.Error())
			}
		})
	}
}

func FuzzRead(f *testing.F) {
	paths, err := filepath.Glob("testdata/reader/*.mrat")
	if err != nil {
		f.Fatal(err)
	}
	paths2, err := filepath.Glob("testdata/reader_error/*.mrat")
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
			expr.String()
		}
	})
}

type runeArray2D struct {
	lines [][]rune
}

func (ra *runeArray2D) String() string {
	var sb strings.Builder
	for _, line := range ra.lines {
		sb.WriteString(string(line))
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (ra *runeArray2D) Set(row, col int, r rune) {
	for row >= len(ra.lines) {
		ra.lines = append(ra.lines, nil)
	}
	if col >= len(ra.lines[row]) {
		spaces := make([]rune, col-len(ra.lines[row])+1)
		for i := range spaces {
			spaces[i] = ' '
		}
		ra.lines[row] = append(ra.lines[row], spaces...)
	}
	ra.lines[row][col] = r
}

func (ra *runeArray2D) SetString(row, col int, s string) {
	for _, r := range s {
		ra.Set(row, col, r)
		col++
	}
}

func printExprAtPosition(ra *runeArray2D, n value.Value) {
	switch v := n.(type) {
	case *value.List:
		start, end := v.Pos(), v.End()
		// special case for quoted values
		if v.Count() == 2 {
			if sym, ok := v.Item().(*value.Symbol); ok && v.End() == v.Next().Item().End() {
				switch sym.Value {
				case "quote":
					ra.Set(start.Line-1, start.Column-1, '\'')
					printExprAtPosition(ra, v.Next().Item())
				case "quasiquote":
					ra.Set(start.Line-1, start.Column-1, '`')
					printExprAtPosition(ra, v.Next().Item())
				case "unquote":
					ra.Set(start.Line-1, start.Column-1, '~')
					printExprAtPosition(ra, v.Next().Item())
				case "splice-unquote":
					ra.SetString(start.Line-1, start.Column-1, "~@")
					printExprAtPosition(ra, v.Next().Item())
				}
				return
			}
		}

		ra.Set(start.Line-1, start.Column-1, '(')
		ra.Set(end.Line-1, end.Column-1, ')')
		ch, cancel := v.Enumerate()
		defer cancel()
		for item := range ch {
			printExprAtPosition(ra, item)
		}
	case *value.Vector:
		start, end := v.Pos(), v.End()
		ra.Set(start.Line-1, start.Column-1, '[')
		ra.Set(end.Line-1, end.Column-1, ']')
		ch, cancel := v.Enumerate()
		defer cancel()
		for item := range ch {
			printExprAtPosition(ra, item)
		}
	case *value.Symbol:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Str:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Char:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Bool:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Keyword:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Num:
		// the exact formatting of the number is not retained, so any test
		// cases that do anything more interesting than integers may fail.
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	case *value.Nil:
		ra.SetString(v.Pos().Line-1, v.Pos().Column-1, v.String())
	default:
		panic(fmt.Sprintf("unexpected node type: %T", v))
	}
}
