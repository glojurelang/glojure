//go:build !glj_aot_runtime

package glj_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

var mathCases = []struct{ name, expression, want string }{
	{"constants", `[m/E m/PI]`, `[2.718281828459045 3.141592653589793]`},
	{"sin", `(m/sin 0)`, `0.0`},
	{"cos", `(m/cos 0)`, `1.0`},
	{"tan", `(m/tan 0)`, `0.0`},
	{"asin", `(m/asin 0)`, `0.0`},
	{"acos", `(m/acos 1)`, `0.0`},
	{"atan", `(m/atan 0)`, `0.0`},
	{"to-radians", `(m/to-radians 180)`, `3.141592653589793`},
	{"to-degrees", `(m/to-degrees m/PI)`, `180.0`},
	{"exp", `(m/exp 0)`, `1.0`},
	{"log", `(m/log 1)`, `0.0`},
	{"log10", `(m/log10 100)`, `2.0`},
	{"sqrt", `(m/sqrt 9)`, `3.0`},
	{"cbrt", `(m/cbrt 27)`, `3.0`},
	{"ieee-remainder", `(m/IEEE-remainder 7 2)`, `-1.0`},
	{"ceil", `(m/ceil 1.2)`, `2.0`},
	{"floor", `(m/floor -1.2)`, `-2.0`},
	{"rint", `(m/rint 2.5)`, `2.0`},
	{"atan2", `(m/atan2 0 1)`, `0.0`},
	{"pow", `(m/pow 2 3)`, `8.0`},
	{"round", `(m/round -1.5)`, `-1`},
	{"random", `(let [r (m/random)] (and (<= 0.0 r) (< r 1.0)))`, `true`},
	{"add-exact", `(m/add-exact 2 3)`, `5`},
	{"subtract-exact", `(m/subtract-exact 2 3)`, `-1`},
	{"multiply-exact", `(m/multiply-exact 2 3)`, `6`},
	{"increment-exact", `(m/increment-exact 2)`, `3`},
	{"decrement-exact", `(m/decrement-exact 2)`, `1`},
	{"negate-exact", `(m/negate-exact 2)`, `-2`},
	{"floor-div", `(m/floor-div -7 3)`, `-3`},
	{"floor-mod", `(m/floor-mod -7 3)`, `2`},
	{"ulp", `(m/ulp 1)`, `2.220446049250313E-16`},
	{"signum", `(m/signum -2)`, `-1.0`},
	{"sinh", `(m/sinh 0)`, `0.0`},
	{"cosh", `(m/cosh 0)`, `1.0`},
	{"tanh", `(m/tanh 0)`, `0.0`},
	{"hypot", `(m/hypot 3 4)`, `5.0`},
	{"expm1", `(m/expm1 0)`, `0.0`},
	{"log1p", `(m/log1p 0)`, `0.0`},
	{"copy-sign", `(m/copy-sign 2 -3)`, `-2.0`},
	{"get-exponent", `(m/get-exponent 8)`, `3`},
	{"next-after", `(m/next-after 1 2)`, `1.0000000000000002`},
	{"next-up", `(m/next-up 1)`, `1.0000000000000002`},
	{"next-down", `(m/next-down 1)`, `0.9999999999999999`},
	{"scalb", `(m/scalb 1.5 3)`, `12.0`},
	{"sqrt-ratio", `(m/sqrt 9/4)`, `1.5`},
	{"transcendental-accuracy", `(let [near? (fn [x expected]
                  (<= (Math/abs (- x expected)) (* 2 (m/ulp expected))))]
              (every? true?
                [(near? (m/sin 1) 0.8414709848078965)
                 (near? (m/cos 1) 0.5403023058681398)
                 (near? (m/tan 1) 1.5574077246549023)
                 (near? (m/asin 0.5) 0.5235987755982989)
                 (near? (m/acos 0.5) 1.0471975511965979)
                 (near? (m/atan 1) 0.7853981633974483)
                 (near? (m/exp 1) 2.718281828459045)
                 (near? (m/log 2) 0.6931471805599453)
                 (near? (m/cbrt 2) 1.2599210498948732)
                 (near? (m/sinh 1) 1.1752011936438014)
                 (near? (m/cosh 1) 1.5430806348152437)
                 (near? (m/tanh 1) 0.7615941559557649)
                 (near? (m/expm1 0.1) 0.10517091807564763)
                 (near? (m/log1p 0.1) 0.09531017980432487)]))`, `true`},
	{"rounding-zero-sign", `(mapv #(Double/doubleToRawLongBits %)
              [(m/rint -0.1) (m/ceil -0.1) (m/IEEE-remainder -4 2)])`,
		`[-9223372036854775808 -9223372036854775808 -9223372036854775808]`},
	{"hinted-long-conversion", `(m/add-exact (Integer/valueOf 2147483647) (Integer/valueOf 1))`, `2147483648`},
	{"hinted-double-conversion", `(let [f m/ulp] (f (float 1)))`, `2.220446049250313E-16`},
	{"hinted-next-up", `(let [f m/next-up] (f (float 1)))`, `1.0000000000000002`},
	{"return-double", `(= (class (m/sqrt 4)) (class 1.0))`, `true`},
	{"return-long", `(= (class (m/add-exact (Integer/valueOf 1) (Integer/valueOf 2))) (class (long 1)))`, `true`},
	{"return-exponent-int", `(= (class (m/get-exponent 1)) (class (Integer/valueOf 1)))`, `true`},
	{"round-special", `[(m/round ##NaN) (m/round ##Inf) (m/round ##-Inf) (m/round 1.5) (m/round -0.5)]`, `[0 9223372036854775807 -9223372036854775808 2 0]`},
	{"nan", `[(Double/isNaN (m/sqrt -1)) (Double/isNaN (m/asin 2)) (Double/isNaN (m/ulp ##NaN)) (Double/isNaN (m/next-after 1 ##NaN))]`, `[true true true true]`},
	{"infinity", `[(m/ulp ##Inf) (m/ulp ##-Inf) (m/next-up ##Inf) (m/next-down ##-Inf)]`, `[##Inf ##Inf ##Inf ##-Inf]`},
	{"exponent-special", `[(m/get-exponent 0.0) (m/get-exponent -0.0) (m/get-exponent 4.9E-324) (m/get-exponent ##Inf) (m/get-exponent ##NaN)]`, `[-1023 -1023 -1023 1024 1024]`},
	{"signed-zero", `[(Double/doubleToRawLongBits (m/sqrt -0.0)) (Double/doubleToRawLongBits (m/signum -0.0)) (Double/doubleToRawLongBits (m/next-after 0.0 -0.0)) (Double/doubleToRawLongBits (m/scalb -0.0 10))]`, `[-9223372036854775808 -9223372036854775808 -9223372036854775808 -9223372036854775808]`},
	{"subnormal", `[(m/ulp 0.0) (m/ulp 4.9E-324) (m/next-up -0.0) (m/next-down 0.0) (m/scalb 1.0 -1074)]`, `[4.9E-324 4.9E-324 4.9E-324 -4.9E-324 4.9E-324]`},
	{"boundaries", `(mapv Double/doubleToRawLongBits [(m/next-after ##Inf 0) (m/next-after ##-Inf 0) (m/ulp 1.7976931348623157E308) (m/next-up 1.7976931348623157E308)])`, `[9218868437227405311 -4503599627370497 8980177656976769024 9218868437227405312]`},
	{"large-scale", `[(m/scalb 1.0 2147483647) (m/scalb 1.0 -2147483648)]`, `[##Inf 0.0]`},
	{"long-min-divide", `[(m/floor-div -9223372036854775808 -1) (m/floor-mod -9223372036854775808 -1)]`, `[-9223372036854775808 0]`},
	{"negative-divisor", `[(m/floor-div 7 -3) (m/floor-mod 7 -3)]`, `[-3 -2]`},
	{"exports", `(count (ns-publics (find-ns (symbol "clojure.math"))))`, `45`},
	{"add-overflow", `(throws? (m/add-exact 9223372036854775807 1))`, `:error`},
	{"subtract-overflow", `(throws? (m/subtract-exact -9223372036854775808 1))`, `:error`},
	{"multiply-overflow", `(throws? (m/multiply-exact 9223372036854775807 2))`, `:error`},
	{"increment-overflow", `(throws? (m/increment-exact 9223372036854775807))`, `:error`},
	{"decrement-overflow", `(throws? (m/decrement-exact -9223372036854775808))`, `:error`},
	{"negate-overflow", `(throws? (m/negate-exact -9223372036854775808))`, `:error`},
	{"divide-zero", `(throws? (m/floor-div 1 0))`, `:error`},
	{"mod-zero", `(throws? (m/floor-mod 1 0))`, `:error`},
}

func mathTestNamespace(t *testing.T, name string) lang.IFn {
	t.Helper()
	lang.PushThreadBindings(lang.NewMap(
		lang.VarCurrentNS, lang.GlobalEnv.CurrentNamespace()))
	t.Cleanup(lang.PopThreadBindings)
	eval := glj.Var("clojure.core", "eval")
	eval.Invoke(glj.Read(fmt.Sprintf(
		"(ns %s (:require [clojure.math :as m]))", name)))
	eval.Invoke(glj.Read("(defmacro throws? [form] " +
		"`(try ~form :no-error (catch go/any _# :error)))"))
	return eval
}

func TestMathOperations(t *testing.T) {
	for _, tc := range mathCases {
		t.Run(tc.name, func(t *testing.T) {
			eval := mathTestNamespace(t, "glojure.test.evaluated-math")
			got := eval.Invoke(glj.Read(tc.expression))
			if !lang.Equals(got, glj.Read(tc.want)) {
				t.Fatalf("%s = %s, want %s",
					tc.expression, lang.PrintString(got), tc.want)
			}
		})
	}
}

func TestCompiledMathOperations(t *testing.T) {
	const namespace = "glojure.test.compiled-math"
	eval := mathTestNamespace(t, namespace)
	for i, tc := range mathCases {
		eval.Invoke(glj.Read(fmt.Sprintf("(defn case-%d [] %s)", i, tc.expression)))
	}
	var generated bytes.Buffer
	if err := runtime.NewGenerator(&generated).Generate(
		lang.FindNamespace(lang.NewSymbol(namespace))); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "generated"), 0755); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	resource := strings.ReplaceAll(strings.ReplaceAll(namespace, ".", "/"), "-", "_")
	files := map[string]string{
		"generated/loader.go": generated.String(),
		"go.mod": fmt.Sprintf("module mathtest\n\ngo 1.24\n\n"+
			"require github.com/glojurelang/glojure v0.0.0\n"+
			"replace github.com/glojurelang/glojure => %s\n", root),
		"main.go": fmt.Sprintf(`package main
import (
    "fmt"
    "github.com/glojurelang/glojure/pkg/glj"
    "github.com/glojurelang/glojure/pkg/lang"
    "github.com/glojurelang/glojure/pkg/runtime"
    _ "github.com/glojurelang/glojure/pkg/stdlib/clojure/math"
    _ "mathtest/generated"
)
func main() {
    runtime.RT.Load("clojure/math")
    runtime.RT.Load(%q)
    values := make([]any, %d)
    for i := range values {
        values[i] = glj.Var(%q, fmt.Sprintf("case-%%d", i)).Invoke()
    }
    fmt.Print(lang.PrintString(lang.NewVector(values...)))
}
`, resource, len(mathCases), namespace),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "run", "-mod=mod", "-tags=glj_aot_runtime", ".")
	command.Dir = directory
	command.Env = append(os.Environ(), "GLOJURE_USE_AOT=true")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled math operations: %v\n%s", err, output)
	}
	results := glj.Read(string(output))
	if lang.Count(results) != len(mathCases) {
		t.Fatalf("compiled result count = %d, want %d", lang.Count(results), len(mathCases))
	}
	for i, tc := range mathCases {
		t.Run(tc.name, func(t *testing.T) {
			got := lang.MustNth(results, i)
			if !lang.Equals(got, glj.Read(tc.want)) {
				t.Fatalf("compiled %s = %s, want %s", tc.name, lang.PrintString(got), tc.want)
			}
		})
	}
}

// Set both variables to compare the shared cases with a JVM Clojure oracle.
// This keeps Java optional for the normal Go test suite.
func TestMathJVMOracle(t *testing.T) {
	java := os.Getenv("GLOJURE_MATH_JAVA")
	classpath := os.Getenv("GLOJURE_MATH_JVM_CLASSPATH")
	if java == "" || classpath == "" {
		t.Skip("set GLOJURE_MATH_JAVA and GLOJURE_MATH_JVM_CLASSPATH")
	}
	source := `(do (require '[clojure.math :as m])
(defmacro throws? [form]
  ` + "`" + `(try ~form :no-error (catch Throwable _# :error)))
(prn [`
	for _, tc := range mathCases {
		source += "\n" + tc.expression
	}
	source += "]))"
	command := exec.Command(java, "-cp", classpath, "clojure.main", "-e", source)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("JVM math oracle: %v\n%s", err, output)
	}
	results := glj.Read(string(output))
	if lang.Count(results) != len(mathCases) {
		t.Fatalf("JVM result count = %d, want %d", lang.Count(results), len(mathCases))
	}
	for i, tc := range mathCases {
		t.Run(tc.name, func(t *testing.T) {
			got := lang.MustNth(results, i)
			if !lang.Equals(got, glj.Read(tc.want)) {
				t.Fatalf("JVM %s = %s, want %s", tc.name, lang.PrintString(got), tc.want)
			}
		})
	}
}
