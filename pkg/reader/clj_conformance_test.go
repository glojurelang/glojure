package reader

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/kylelemons/godebug/diff"
	"github.com/stretchr/testify/assert"
)

type (
	testSymbolResolver struct{}
)

func (sr *testSymbolResolver) CurrentNS() *value.Symbol {
	return value.NewSymbol("user")
}

func (sr *testSymbolResolver) ResolveStruct(s *value.Symbol) *value.Symbol {
	if strings.Contains(s.Name(), ".") {
		return s
	}
	return nil
}

func (sr *testSymbolResolver) ResolveAlias(s *value.Symbol) *value.Symbol {
	return s
}

func (sr *testSymbolResolver) ResolveVar(s *value.Symbol) *value.Symbol {
	if strings.Contains(s.String(), "/") {
		return s
	}
	return value.NewSymbol("user/" + s.String())
}

// Running these fuzz tests is slow because clj is very slow to start
// up. Use GLOJ_WRITE_GLJ_FUZZ_TEST_CACHE=1 to cache the output of
// clj. DO NOT use this when actually fuzzing, because it will
// generate many unnecessary files.
//
// TODOs:
// - automatically detect if we're fuzzing, and ignore the env var if so
// - use a pool of clj processes instead of starting a new one for each test
func FuzzCLJConformance(f *testing.F) {
	paths, err := filepath.Glob("testdata/reader/*.glj")
	if err != nil {
		f.Fatal(err)
	}
	for _, path := range paths {
		if strings.Contains(path, "quasi") {
			// skip quasiquote tests for now
			continue
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(string(data))
	}

	cljRdr := newCLJReader()
	// go cljRdr.start()
	// defer cljRdr.stop()

	f.Fuzz(func(t *testing.T, program string) {
		// reject program strings with non-ascii or non-printable
		// characters this prevents the fuzzer from generating exotic
		// unicode while we're still trying to get the basics working.
		for _, c := range program {
			if (c > unicode.MaxASCII || !unicode.IsPrint(c)) && !unicode.IsSpace(c) {
				t.Skipf("program includes non-ascii character: %q", c)
			}
		}

		t.Logf("program (quoted): %q", program)
		t.Logf("program: %s", program)

		cljExpr, cljErr := cljRdr.readCLJExpr(program)

		r := New(strings.NewReader(program), WithSymbolResolver(&testSymbolResolver{}))
		// we only want the first expression. TODO: variant that reads
		// one.
		gljValue, gljErr := r.ReadOne()

		if (cljErr == nil) != (gljErr == nil) {
			if isCLJConformanceErrorException(cljErr, cljExpr) {
				t.Skipf("clj error: %v", cljErr)
			}
			t.Logf("clj out: %v", cljExpr)
			t.Fatalf("error mismatch: cljErr=%v gljErr=%v", cljErr, gljErr)
		}
		if cljErr != nil {
			return
		}

		gljExpr := cljNormalize(testPrintString(gljValue))

		// workaround for the fact that Go is able to quote more
		// unprintable characters than Clojure. e.g. \x00 and \x10.
		cljExprRunes := make([]rune, 0, len(cljExpr))
		for _, r := range cljExpr {
			quoted := strconv.QuoteRune(r)
			if strings.HasPrefix(quoted, "'\\x") {
				cljExprRunes = append(cljExprRunes, []rune(quoted[1:len(quoted)-1])...)
			} else {
				cljExprRunes = append(cljExprRunes, r)
			}
		}
		cljExpr = string(cljExprRunes)

		if gljExpr != cljExpr {
			assert.Equal(t, []rune(cljExpr), []rune(gljExpr))
			t.Errorf("want len=%d, got len=%d", len(cljExpr), len(gljExpr))
			t.Errorf("diff (-want,+got):\n%s", diff.Diff(cljExpr, gljExpr))
			t.Fatalf("expression mismatch: glj=%v clj=%v", gljExpr, cljExpr)
		}
	})
}

// isCLJConformanceErrorException returns true if the error is one that
// we expect to see in the clj conformance tests.
func isCLJConformanceErrorException(cljErr error, cljOut string) bool {
	if cljErr == nil {
		return false
	}

	// Java and Go both support regex repetitions in braces (e.g. "a{n,
	// m}" for multiple "a"s), but Go's regex engine is more permissive
	// and will interpret braces as literals if they don't match the
	// regex repetition syntax.
	if strings.Contains(cljOut, "Execution error (PatternSyntaxException) at java.util.regex.Pattern/error") && strings.Contains(cljOut, "Illegal repetition") {
		return true
	}
	return false
}

type cljReaderCommand struct {
	cmd    *exec.Cmd
	pipeIn io.WriteCloser
	out    string
	err    error
	done   chan struct{}
}

// cljReader keeps a pool of clj processes ready to read expressions. This is
// needed because clj is very slow to spin up.
type cljReader struct {
	commands chan *cljReaderCommand
	stopCh   chan struct{}
}

func newCLJReader() *cljReader {
	return &cljReader{
		commands: make(chan *cljReaderCommand, 32),
		stopCh:   make(chan struct{}),
	}
}

func (r *cljReader) start() {
	for {
		// Run the equivalent of the following shell command:
		// clj -M -e '(pr (read *in*))'
		cmd := exec.Command("clj", "-M", "-e", "(pr (read *in*))")
		pipeIn, err := cmd.StdinPipe()
		if err != nil {
			panic(err)
		}

		rdrCommand := &cljReaderCommand{
			cmd:    cmd,
			pipeIn: pipeIn,
			done:   make(chan struct{}),
		}
		go func() {
			out, err := cmd.CombinedOutput()
			rdrCommand.out = string(out)
			rdrCommand.err = err
			close(rdrCommand.done)
		}()

		select {
		case r.commands <- rdrCommand:
		case <-r.stopCh:
			pipeIn.Close()
			return
		}
	}
}

func (r *cljReader) stop() {
	close(r.stopCh)
}

// getFromCache looks for a file in testdata/clj-cache/read with the
// name <hash>.glj where <hash> is the sha256 hash of the program. If
// the file exists, its first line contains "true" if clj was able to
// read the program, and "false" otherwise. The rest of the file
// contains the output of clj.
func getFromCache(program string) (bool, string, error) {
	hash := sha256.Sum256([]byte(program))
	path := filepath.Join("testdata", "clj-cache", "read", fmt.Sprintf("%x.glj", hash))
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return false, "", nil
	}
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) != 2 {
		return false, "", fmt.Errorf("invalid cache file: %s", path)
	}
	switch lines[0] {
	case "true":
		err = nil
	case "false":
		err = errors.New("clj error")
	default:
		return false, "", fmt.Errorf("invalid cache file: %s", path)
	}
	return true, lines[1], err
}

// putToCache writes the result of a clj read to the cache.
func putToCache(program string, err error, out string) error {
	hash := sha256.Sum256([]byte(program))
	path := filepath.Join("testdata", "clj-cache", "read", fmt.Sprintf("%x.glj", hash))
	// if it already exists, don't overwrite it.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	var errStr string
	if err != nil {
		errStr = "false"
	} else {
		errStr = "true"
	}
	return ioutil.WriteFile(path, []byte(errStr+"\n"+out), 0644)
}

func (r *cljReader) readCLJExpr(program string) (string, error) {
	if ok, out, err := getFromCache(program); ok {
		return out, err
	} else if err != nil {
		panic(err)
	}

	// Run the equivalent of the following shell command:
	// clj -M -e '(pr (read *in*))'
	//
	// with program as stdin.
	cmd := exec.Command("clj", "-M", "-e", "(pr (read *in*))")
	cmd.Stdin = strings.NewReader(program)
	out, err := cmd.CombinedOutput()

	if os.Getenv("GLOJ_WRITE_GLJ_FUZZ_TEST_CACHE") != "" {
		if err := putToCache(program, err, string(out)); err != nil {
			panic(err)
		}
	}

	return string(out), err

	// rdrCommand := <-r.commands
	// rdrCommand.pipeIn.Write([]byte(program))
	// rdrCommand.pipeIn.Close()
	// <-rdrCommand.done
	// return rdrCommand.out, rdrCommand.err
}

// for later: clj -M -e '(binding [*print-meta* true] (pr (read *in*)))'

func cljNormalize(s string) string {
	// replace glojure with clojure
	s = strings.ReplaceAll(s, "glojure", "clojure")
	return s
}
