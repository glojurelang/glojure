//go:build !glj_aot_runtime

package runtime

import (
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
)

func TestEvalWithASTDumpUsesDumpedTree(t *testing.T) {
	form, err := reader.New(
		strings.NewReader("(let [x 1] (+ x 2))"),
		reader.WithGetCurrentNS(lang.GlobalEnv.CurrentNamespace),
	).ReadOne()
	if err != nil {
		t.Fatal(err)
	}

	var dump strings.Builder
	got, err := EvalWithASTDump(form, &dump)
	if err != nil {
		t.Fatal(err)
	}
	if !lang.Equals(got, int64(3)) {
		t.Fatalf("EvalWithASTDump() = %v, want 3", got)
	}
	for _, want := range []string{"(let", ":bindings", "(host-call", ":args", "(local"} {
		if !strings.Contains(dump.String(), want) {
			t.Errorf("dump missing %q:\n%s", want, dump.String())
		}
	}
	if strings.Contains(dump.String(), "0x") {
		t.Fatalf("dump contains an unstable address:\n%s", dump.String())
	}
}

func TestSetASTDumpWriterCoversOrdinaryEval(t *testing.T) {
	var dump strings.Builder
	restore, err := SetASTDumpWriter(&dump)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	got, err := lang.GlobalEnv.Eval(int64(7))
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(7) {
		t.Fatalf("Eval() = %v, want 7", got)
	}
	if !strings.Contains(dump.String(), "(const") ||
		!strings.Contains(dump.String(), ":value 7") {
		t.Fatalf("ordinary Eval did not emit its AST:\n%s", dump.String())
	}
}
