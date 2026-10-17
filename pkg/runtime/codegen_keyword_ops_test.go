//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestCompiledKeywordOperationsPreserveOrderAndRedefinition(t *testing.T) {
	const namespace = "codegen.keyword-operations"
	ns := lang.FindOrCreateNamespace(lang.NewSymbol(namespace))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()
	ReadEval(`
        (defn choose [x] (case x :a :found :b :second :default))
        (defn drop-keys [m] (dissoc m :a :b))
        (defn full-match [s] (re-matches #"a|ab" s))
        (defn collect [f xs] (vec (map f xs)))
 (defn entries [m] (vec (mapcat seq m)))
        (defn check []
          (let [events (atom [])
                result (dissoc
                         (do (swap! events conj :map) {:a 1 :b 2 :c 3})
                         (do (swap! events conj :a) :a)
                         (do (swap! events conj :b) :b))]
            [(choose :a) (choose :missing) (choose "a") result @events
             (with-redefs [clojure.core/dissoc (fn [& args] :redefined)]
               (drop-keys {:a 1 :b 2}))
             (full-match "ab")
             (re-matches #"(a|ab)(b?)" "ab")
             (re-matches #"(?m)[^\n]+" "line\n")
             (re-matches #"" "")
             (with-redefs [clojure.core/re-matches (fn [& args] :redefined)]
               (full-match "ab"))
             (collect inc [1 2 3])
             (with-redefs [clojure.core/map (fn [f xs] [:changed])]
               (collect inc [1 2]))
             (with-redefs [clojure.core/vec (fn [xs] :changed)]
               (collect inc [1 2]))
             (let [events (atom [])
                   result (collect
                            (do (swap! events conj :fn)
                                (fn [x] (swap! events conj x) x))
                            (do (swap! events conj :source) [1 2]))]
               [result @events])
             (entries {:a 1})
             (with-redefs [clojure.core/seq (fn [x] [:changed])]
               (entries {:a 1}))
             (with-redefs [clojure.core/mapcat (fn [f xs] [:changed])]
               (entries {:a 1}))]))`)
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprintf("direct-link=%v", direct), func(t *testing.T) {
			runCompiledLiteralOperations(t, ns, direct)
		})
	}
}

func runCompiledLiteralOperations(t *testing.T, ns *lang.Namespace, direct bool) {
	t.Helper()
	var output bytes.Buffer
	if err := newGenerator(&output, direct).Generate(ns); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "runtime.VectorMapcatSeq(") {
		t.Fatal("compiler did not fuse vec of mapcat seq")
	}
	if !strings.Contains(output.String(), "runtime.VectorMap(") {
		t.Fatal("compiler did not fuse vec of map")
	}
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "generated"), 0755); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"generated/loader.go": output.String(),
		"go.mod": fmt.Sprintf("module keywordtest\n\ngo 1.24\n\n"+
			"require github.com/glojurelang/glojure v0.0.0\n"+
			"replace github.com/glojurelang/glojure => %s\n", root),
		"main.go": `package main
import (
    "fmt"
    "github.com/glojurelang/glojure/pkg/glj"
    "github.com/glojurelang/glojure/pkg/lang"
    _ "keywordtest/generated"
)
func main() {
    glj.Var("clojure.core", "require").Invoke(lang.NewSymbol("codegen.keyword-operations"))
    fmt.Print(lang.PrintString(glj.Var("codegen.keyword-operations", "check").Invoke()))
}
`,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "run", "-mod=mod", "-tags=glj_no_goimports", ".")
	command.Dir = directory
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled keyword operations: %v\n%s", err, result)
	}
	want := `[:found :default :default {:c 3} [:map :a :b] :redefined "ab" ["ab" "a" "b"] nil "" :redefined [2 3 4] [:changed] :changed [[1 2] [:fn :source 1 2]] [:a 1] [:changed] [:changed]]`
	if direct {
		want = `[:found :default :default {:c 3} [:map :a :b] {} "ab" ["ab" "a" "b"] nil "" "ab" [2 3 4] [2 3] [2 3] [[1 2] [:fn :source 1 2]] [:a 1] [:changed] [:a 1]]`
	}
	if got := strings.TrimSpace(string(result)); got != want {
		t.Fatalf("compiled operations = %s, want %s", got, want)
	}
}
