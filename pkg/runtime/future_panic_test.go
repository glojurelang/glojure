package runtime_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core"
)

// A regression in AgentSubmit must kill only the child, not the test runner.
// These expressions use the original core macros and AOT pmap, without Var overrides.
func TestFuturePanicSubprocess(t *testing.T) {
	cases := map[string]string{
		"future":        `(= :caught (try @(future (throw (ex-info "probe" {:value :caught}))) (catch Exception e (:value (ex-data e)))))`,
		"pmap":          `(= :caught (try (doall (pmap (fn [x] (throw (ex-info "probe" {:value :caught}))) [1])) (catch Exception e (:value (ex-data e)))))`,
		"repeat":        `(let [f (future (throw (ex-info "probe" {:value 42}))) read-error (fn [] (try @f (catch Exception e (:value (ex-data e)))))] (= [42 42 true true] [(read-error) (read-error) (realized? f) (future? f)]))`,
		"timeout":       `(let [gate (promise) f (future @gate) v (deref f 5 :timeout)] (deliver gate 42) (= [:timeout 42] [v @f]))`,
		"timeout-error": `(let [gate (promise) f (future @gate (throw (ex-info "probe" {}))) v (deref f 5 :timeout)] (deliver gate true) (= [:timeout :caught] [v (try (deref f 10000 :late) (catch Exception e :caught))]))`,
		"stdout":        `(do @(future (println "future output")) true)`,
		"bindings":      `(= [2 2] (let [f (binding [*print-length* 2] (future [*print-length* @(future *print-length*)]))] @f))`,
		"pmap-success":  `(= [2 3 4] (vec (pmap inc [1 2 3])))`,
	}
	if name := os.Getenv("GLJ_FUTURE_PANIC_CHILD"); name != "" {
		source, ok := cases[name]
		if !ok {
			t.Fatal(name)
		}
		var output strings.Builder
		env := runtime.NewEnvironment(runtime.WithStdout(&output))
		lang.PushThreadBindings(lang.NewMap(lang.VarOut, &output))
		defer lang.PopThreadBindings()
		if got := runtime.ReadEval(source, runtime.WithEnv(env)); got != true {
			t.Fatalf("result: %v", got)
		}
		if name == "stdout" && output.String() != "future output\n" {
			t.Fatalf("stdout: %q", output.String())
		}
		if got := runtime.ReadEval(`(= 42 (+ 20 22))`, runtime.WithEnv(env)); got != true {
			t.Fatal("environment did not survive")
		}
		return
	}
	for name := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFuturePanicSubprocess$")
			cmd.Env = append(os.Environ(), "GLJ_FUTURE_PANIC_CHILD="+name)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("child: %v (context: %v)\n%s", err, ctx.Err(), out)
			}
		})
	}
}

func TestFuturePanicAccessors(t *testing.T) {
	failure := &struct{ message string }{"original panic"}
	f := lang.AgentSubmit(lang.FnFunc0(func() any { panic(failure) }))
	readers := []func() any{
		f.(lang.IDeref).Deref,
		func() any { return f.DerefWithTimeout(10000, nil) },
		f.(lang.Future).Get,
		func() any { return f.(lang.Future).GetWithTimeout(10000, time.Millisecond) },
	}
	for i, read := range readers {
		func() {
			defer func() {
				if got := recover(); got != failure {
					t.Errorf("accessor %d: panic %v", i, got)
				}
			}()
			read()
			t.Errorf("accessor %d did not panic", i)
		}()
	}
	if !f.(lang.IPending).IsRealized() {
		t.Fatal("failed future not realized")
	}
}
