package reader

import (
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/kylelemons/godebug/diff"
)

var (
	testSymbolResolver = SymbolResolverFunc(func(s string) string {
		if strings.Contains(s, "/") {
			return s
		}
		return "user/" + s
	})
)

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
		// reject program strings with non-printable characters
		for _, c := range program {
			if !unicode.IsPrint(c) {
				t.Skipf("program includes non-printable character: %q", c)
			}
		}

		t.Logf("program (quoted): %q", program)
		t.Logf("program: %s", program)

		cljExpr, cljErr := cljRdr.readCLJExpr(program)

		r := New(strings.NewReader(program), WithSymbolResolver(testSymbolResolver))
		// we only want the first expression. TODO: variant that reads
		// one.
		gljValue, gljErr := r.ReadOne()

		if (cljErr == nil) != (gljErr == nil) {
			t.Logf("clj out: %v", cljExpr)
			t.Fatalf("error mismatch: cljErr=%v gljErr=%v", cljErr, gljErr)
		}
		if cljErr != nil {
			return
		}

		gljExpr := gljValue.String()

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
			t.Errorf("want len=%d, got len=%d", len(cljExpr), len(gljExpr))
			t.Errorf("diff (-want,+got):\n%s", diff.Diff(cljExpr, gljExpr))
			t.Fatalf("expression mismatch: glj=%v clj=%v", gljExpr, cljExpr)
		}
	})
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

func (r *cljReader) readCLJExpr(program string) (string, error) {
	// Run the equivalent of the following shell command:
	// clj -M -e '(pr (read *in*))'
	//
	// with program as stdin.
	cmd := exec.Command("clj", "-M", "-e", "(pr (read *in*))")
	cmd.Stdin = strings.NewReader(program)
	out, err := cmd.CombinedOutput()
	return string(out), err

	// rdrCommand := <-r.commands
	// rdrCommand.pipeIn.Write([]byte(program))
	// rdrCommand.pipeIn.Close()
	// <-rdrCommand.done
	// return rdrCommand.out, rdrCommand.err
}

// for later: clj -M -e '(binding [*print-meta* true] (pr (read *in*)))'
