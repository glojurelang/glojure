//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"io/fs"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/google/uuid"
)

func TestGenerateNamedScalarValue(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})

	got := generator.generateValue(fs.ModeSymlink)
	if want := "fs0.FileMode(134217728)"; got != want {
		t.Fatalf("generated fs.FileMode = %q, want %q", got, want)
	}
	if got := generator.imports["io/fs"]; got != "fs0" {
		t.Fatalf("io/fs import alias = %q, want %q", got, "fs0")
	}

	id := uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")
	got = generator.generateValue(id)
	if !strings.HasPrefix(got, "uuid1.UUID{") {
		t.Fatalf("generated uuid.UUID = %q", got)
	}
	if got := generator.imports["github.com/google/uuid"]; got != "uuid1" {
		t.Fatalf("uuid import alias = %q, want %q", got, "uuid1")
	}
}

func TestGenerateInferredPrimitiveFunctionValues(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.primitive-function-values"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(def pipeline
		  (filter (fn [value] (not (even? value)))))
		(defn build []
		  [(map (fn [value] (* value value)))
		   (filter (fn [value] (zero? (mod value 3))))
		   (filter odd?)])`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatal(err)
	}
	source := output.String()
	if !strings.Contains(source, "lang.NewInt64UnaryFn(") {
		t.Fatalf("generated source omitted primitive unary entry point:\n%s", source)
	}
	if got := strings.Count(
		source,
		"lang.NewInt64PredicateFn(",
	); got < 2 {
		t.Fatalf(
			"generated source has %d primitive predicate entry points, want at least 2:\n%s",
			got,
			source,
		)
	}

	output.Reset()
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "lang.NewInt64") {
		t.Fatalf(
			"direct-link-disabled source contains primitive callable entry point:\n%s",
			output.String(),
		)
	}
}

func TestGenerateNativeTransducerValues(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})
	mapper := lang.NewKeyword("value")

	mapExpression := generator.generateValue(NewMapTransducer(mapper))
	if !strings.Contains(mapExpression, "runtime.NewMapTransducer(") {
		t.Fatalf("generated map transducer = %q", mapExpression)
	}
	takeExpression := generator.generateValue(
		NewTakeTransducer(int64(17)),
	)
	if takeExpression != "runtime.NewTakeTransducer(int64(17))" {
		t.Fatalf("generated take transducer = %q", takeExpression)
	}

	withMeta := NewFilterTransducer(mapper).(lang.IObj).WithMeta(
		lang.NewMap(lang.NewKeyword("line"), int64(12)),
	)
	filterExpression := generator.generateValue(withMeta)
	if !strings.Contains(
		filterExpression,
		").(lang.IObj).WithMeta(",
	) {
		t.Fatalf(
			"generated metadata-bearing filter transducer = %q",
			filterExpression,
		)
	}
}

func TestGenerateStandardFileHandles(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})

	for handle, want := range map[*os.File]string{
		os.Stdin:  "os0.Stdin",
		os.Stdout: "os0.Stdout",
		os.Stderr: "os0.Stderr",
	} {
		if got := generator.generateValue(handle); got != want {
			t.Errorf("generated standard file handle = %q, want %q", got, want)
		}
	}
}

func TestGenerateArbitraryPrecisionConstants(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})

	bigInt, err := lang.NewBigInt("123456789012345678901234567890")
	if err != nil {
		t.Fatal(err)
	}
	if got := generator.generateValue(bigInt); !strings.Contains(
		got,
		`lang.NewBigInt("123456789012345678901234567890")`,
	) {
		t.Fatalf("generated BigInt = %q", got)
	}

	ratio := lang.NewRatioBigInt(
		lang.NewBigIntFromInt64(-1),
		lang.NewBigIntFromInt64(5),
	)
	got := generator.generateValue(ratio)
	for _, want := range []string{`lang.NewBigInt("-1")`, `lang.NewBigInt("5")`} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated Ratio = %q, missing %q", got, want)
		}
	}
}

func TestGenerateNonFiniteFloatConstants(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})

	for name, value := range map[string]float64{
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
		"not a number":      math.NaN(),
		"negative zero":     math.Copysign(0, -1),
	} {
		got := generator.generateValue(value)
		if !strings.Contains(got, "math0.Float64frombits(") {
			t.Errorf("generated %s = %q, want exact IEEE-754 reconstruction", name, got)
		}
	}
	if got := generator.imports["math"]; got != "math0" {
		t.Fatalf("math import alias = %q, want %q", got, "math0")
	}
}

func TestGenerateRegexConstantsPreserveOccurrenceIdentity(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})
	first := regexp.MustCompile("same pattern")
	second := regexp.MustCompile("same pattern")

	constant := func(value *regexp.Regexp) *ast.Node {
		node := ast.MakeNode(ast.OpConst, nil)
		node.Sub = &ast.ConstNode{Value: value}
		return node
	}

	firstName := generator.generateASTNode(constant(first))
	if got := generator.generateASTNode(constant(first)); got != firstName {
		t.Fatalf("same regex object generated as %q then %q", firstName, got)
	}
	if secondName := generator.generateASTNode(constant(second)); secondName == firstName {
		t.Fatalf("distinct regex literals shared generated constant %q", firstName)
	}
	if got := generator.generateValue(first); !strings.Contains(got, ".MustCompile(") {
		t.Fatalf("generated regex initializer = %q, want regexp.MustCompile", got)
	}
}

func TestMungePackageNameAvoidsGoKeywords(t *testing.T) {
	for input, want := range map[string]string{
		"case":   "pkg_case",
		"normal": "normal",
		"1thing": "pkg_1thing",
	} {
		if got := mungePackageName(input); got != want {
			t.Errorf("mungePackageName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateNestedClosureCapturedAtLoadTime(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.nested-capture"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn make-closure [keep]
		  (fn [_]
		    (let [finish (fn [] keep)]
		      (finish))))
		(def captured (make-closure true))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate nested captured closure: %v", err)
	}
}

func TestGenerateOwnedLoopMapUsesTransientRepresentation(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.owned-loop-map"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn histogram [values]
		  (loop [remaining (seq values)
		         counts {}]
		    (if remaining
		      (let [value (first remaining)]
		        (recur (next remaining)
		               (assoc counts value (inc (get counts value 0)))))
		      counts)))
		(defn escaping [values observe]
		  (loop [remaining (seq values)
		         counts {}]
		    (if remaining
		      (do
		        (observe counts)
		        (recur (next remaining)
		               (assoc counts (first remaining) 1)))
		      counts)))
		(defn update-map [values]
		  (loop [index 0
		         result values]
		    (if (= index 2)
		      result
		      (recur (inc index)
		             (assoc result index index)))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate owned map loop: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		".(lang.IEditableCollection).AsTransient()",
		".(*lang.TransientMap).Assoc(",
		".(*lang.TransientMap).Persistent()",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("owned map lowering omitted %q:\n%s", expected, generated)
		}
	}
	if got := strings.Count(
		generated,
		".(lang.IEditableCollection).AsTransient()",
	); got != 1 {
		t.Fatalf(
			"generated %d transient map regions, want only non-escaping histogram",
			got,
		)
	}
	for _, expected := range []string{
		"runtime.NewOwnedLoopMap(",
		".(*runtime.OwnedLoopMap).Assoc(",
		".(*runtime.OwnedLoopMap).Persistent()",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("adaptive owned map lowering omitted %q:\n%s",
				expected, generated)
		}
	}
}

func TestGenerateDirectLinkedInt64VectorUpdateRegion(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.vector-update-region"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn exchange [values left right]
		  (let [left-value (nth values left)
		        right-value (nth values right)]
		    (assoc (assoc values left right-value) right left-value)))
		(defn reverse-owned [values length]
		  (loop [result values left 0 right (dec length)]
		    (if (>= left right)
		      result
		      (recur (exchange result left right)
		             (inc left)
		             (dec right)))))
		(defn flip-score [values]
		  (loop [owned values score 0]
		    (let [first-value (nth owned 0)]
		      (if (zero? first-value)
		        score
		        (recur (reverse-owned owned (inc first-value))
		               (inc score))))))`)

	var output bytes.Buffer
	generator := newGenerator(&output, true)
	if err := generator.Generate(ns); err != nil {
		t.Fatalf("generate vector update region: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"lang.CanTransientlyUpdateInt64Vector(",
		".AsTransientForUpdate()",
		"*lang.TransientVector",
		".AssocN(",
		".Persistent()",
		"aotVectorFn",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("vector update lowering omitted %q:\n%s", expected, generated)
		}
	}
	if !strings.Contains(generated, "lang.Assoc(") {
		t.Fatalf("vector specialization omitted the dynamic fallback:\n%s", generated)
	}

	output.Reset()
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate without direct linking: %v", err)
	}
	if strings.Contains(output.String(), "aotVectorFn") {
		t.Fatalf(
			"vector specialization ignored disabled direct linking:\n%s",
			output.String(),
		)
	}
}

func TestVectorUpdateRegionOnlyFreezesAtTerminalEscape(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		source     string
		specialize bool
	}{
		{
			name:      "terminal escape",
			namespace: "codegen.vector-terminal-escape",
			source: `
				(defn update-and-return [values]
				  [(assoc values 0 1)])`,
			specialize: true,
		},
		{
			name:      "early escape",
			namespace: "codegen.vector-early-escape",
			source: `
				(defn expose-then-update [values]
				  (do
				    [values]
				    (assoc values 0 1)))`,
			specialize: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns := lang.FindOrCreateNamespace(lang.NewSymbol(test.namespace))
			ns.ReferAllSnapshot(lang.NSCore, nil)
			lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
			defer lang.PopThreadBindings()
			ReadEval(test.source)

			var output bytes.Buffer
			if err := newGenerator(&output, true).Generate(ns); err != nil {
				t.Fatalf("generate: %v", err)
			}
			generated := output.String()
			if got := strings.Contains(
				generated,
				".AsTransientForUpdate()",
			); got != test.specialize {
				t.Fatalf(
					"specialization present = %v, want %v:\n%s",
					got,
					test.specialize,
					generated,
				)
			}
		})
	}
}

func TestGenerateTypedStringOwnedMapOperations(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.typed-string-map"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn substring-histogram [text]
		  (loop [i 0
		         counts {}]
		    (if (= i (count text))
		      counts
		      (let [token (subs text i (inc i))]
		        (recur (inc i)
		               (assoc counts token
		                      (inc (get counts token 0))))))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate typed string map loop: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		" int64 = ",
		" string = ",
		".ValAtStringDefault(",
		".AssocString(",
		"runtime.RT.SubsEnd(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("typed lowering omitted %q:\n%s", expected, generated)
		}
	}
}

func TestGenerateFixedArityFunctionsThroughTwenty(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.fixed-arity-twenty"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn fixed
		  [a b c d e f g h i j k l m n o p q r s t]
		  [a b c d e f g h i j k l m n o p q r s t])
		(defn invoke [f]
		  (f 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate fixed arity function: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "lang.FnFunc20") {
		t.Fatalf("twenty-argument function did not use FnFunc20:\n%s", generated)
	}
	if !strings.Contains(generated, "lang.Apply20") {
		t.Fatalf("dynamic twenty-argument call did not use Apply20:\n%s", generated)
	}
}

func TestGenerateConcreteRecordTypeAndDirectOperations(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.record"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defrecord Point [x y])
		(defn move [point x y]
		  (assoc point :x x :y y))
		(defn x-coordinate [point]
		  (:x point))
		(defn label [value]
		  (:label value))
		(defn tag [value tag]
		  (assoc value :tag tag))
		(defn origin []
		  (Point. 0 0))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate record: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"type aotRecord0Point struct",
		"attrs *lang.RecordAttrs",
		"func aotRecordNew0(",
		"case *aotRecord0Point:",
		"return value.f0",
		"case *aotRecord0Point:",
		"result.f0 = v0",
		"result.f1 = v1",
		":= aotRecordNew0(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("generated record omitted %q:\n%s", expected, generated)
		}
	}
	if strings.Contains(generated, "type aotKeywordMap") {
		t.Fatalf("record generation revived arbitrary map specialization:\n%s",
			generated)
	}
	for _, unwanted := range []string{
		"\n\tmeta lang.IPersistentMap\n",
		"\n\text lang.IPersistentMap\n",
		"\n\thash uint32\n",
		"\n\thasheq uint32\n",
	} {
		if strings.Contains(generated, unwanted) {
			t.Fatalf("generated record retained eager %q state:\n%s",
				unwanted, generated)
		}
	}
	if got := strings.Count(generated, "func aotKeywordLookup"); got != 1 {
		t.Fatalf("generated %d record lookup helpers, want 1:\n%s",
			got, generated)
	}
	if got := strings.Count(generated, "func aotKeywordAssoc"); got != 1 {
		t.Fatalf("generated %d record assoc helpers, want 1:\n%s",
			got, generated)
	}
}

func TestGenerateRecursiveRecordSpecialization(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.record-specialization"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defrecord Node [value left right])
		(defn make-node [value depth]
		  (if (zero? depth)
		    (->Node value nil nil)
		    (let [next-depth (dec depth)]
		      (->Node value
		              (make-node (dec value) next-depth)
		              (make-node (inc value) next-depth)))))
		(defn sum-node [node]
		  (if (nil? (:left node))
		    (:value node)
		    (+ (:value node)
		       (sum-node (:left node))
		       (sum-node (:right node)))))
		(defn equal-node-score [left right]
		  (if (= left right) 1 0))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate specialized record: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"func aotRecordFastNew0(",
		"f0    int64",
		"f1    *aotRecord0Node",
		"f2    *aotRecord0Node",
		"var aotRecordFn",
		".aotRecordFast()",
		"= aotRecordFastNew0(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("record specialization omitted %q:\n%s",
				expected, generated)
		}
	}
	if got := strings.Count(generated, "var aotRecordFn"); got != 2 {
		t.Fatalf(
			"generated %d record-specialized functions, want 2; "+
				"record equality must retain generic semantics:\n%s",
			got,
			generated,
		)
	}

	output.Reset()
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate non-direct record: %v", err)
	}
	if strings.Contains(output.String(), "aotRecordFastNew") ||
		strings.Contains(output.String(), "aotRecordFn") {
		t.Fatalf("record specialization ignored disabled direct linking:\n%s",
			output.String())
	}

	makeNode := ns.FindInternedVar(lang.NewSymbol("make-node"))
	originalMeta := makeNode.Meta()
	makeNode.SetMeta(originalMeta.Assoc(lang.KWRedef, true).(lang.IPersistentMap))
	defer makeNode.SetMeta(originalMeta)

	output.Reset()
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate redef record producer: %v", err)
	}
	if strings.Contains(output.String(), "aotRecordFastNew") ||
		strings.Contains(output.String(), "aotRecordFn") {
		t.Fatalf("record specialization ignored ^:redef producer:\n%s",
			output.String())
	}
}

func TestGenerateBooleanRecordSpecialization(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.record-bool-specialization"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defrecord Flag [enabled])
		(defn make-flag [value]
		  (if (zero? value)
		    (->Flag true)
		    (->Flag false)))
		(defn flag-score [flag]
		  (if (:enabled flag) 1 0))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate boolean record specialization: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"func aotRecordFastNew0(p0 bool)",
		"f0    bool",
		".f0",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("boolean record specialization omitted %q:\n%s",
				expected, generated)
		}
	}
}

func TestGenerateDirectProtocolLinkCanBeDisabled(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.direct-protocol"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defprotocol Combine
		  (combine [target left right]))
		(extend-protocol Combine
		  nil
		  (combine [_ left right] (+ left right)))
		(defn call-combine [left right]
		  (combine nil left right))
		(defn combine-loop []
		  (loop [i 0
		         total 0]
		    (if (= i 100)
		      total
		      (recur (inc i) (combine nil total i)))))
		(defn combine-float-loop []
		  (let [result
		        (loop [i 0
		               value 0.25
		               checksum 0.0]
		          (if (= i 100)
		            [value checksum]
		            (let [input (* 0.000001 (- (mod i 20) 10))
		                  next-value (combine nil value input)]
		              (recur (inc i)
		                     next-value
		                     (+ checksum next-value)))))
		        value (nth result 0)
		        checksum (nth result 1)]
		    [(long (* 1000000.0 value))
		     (long checksum)]))
		(defn mutate-and-call []
		  (extend-protocol Combine
		    nil
		    (combine [_ left right] (* left right)))
		  (combine nil 6 7))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate direct protocol: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "var aotProtocolFn0 *lang.MultiFn") ||
		!strings.Contains(generated, "aotProtocolFn0.Invoke3(") ||
		!strings.Contains(generated,
			"aotProtocolFn0.ProtocolGeneration() == aotProtocolGeneration0") ||
		!strings.Contains(generated, "aotProtocolMethod") {
		t.Fatalf("direct protocol target was not generated:\n%s", generated)
	}
	if got := strings.Count(
		generated,
		"aotProtocolFn0.ProtocolGeneration() == aotProtocolGeneration0",
	); got != 2 {
		t.Fatalf(
			"generated %d guarded protocol regions, want both pure loops:\n%s",
			got,
			generated,
		)
	}
	if !strings.Contains(generated, "aotProtocolMethod3For0(nil") {
		t.Fatalf("float protocol loop retained boxed dispatch:\n%s", generated)
	}

	output.Reset()
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate non-direct protocol: %v", err)
	}
	if strings.Contains(output.String(), "aotProtocolFn") {
		t.Fatalf("protocol direct linking ignored disabled direct linking:\n%s",
			output.String())
	}

	combine := ns.FindInternedVar(lang.NewSymbol("combine"))
	originalMeta := combine.Meta()
	combine.SetMeta(originalMeta.Assoc(lang.KWRedef, true).(lang.IPersistentMap))
	defer combine.SetMeta(originalMeta)

	output.Reset()
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate redef protocol: %v", err)
	}
	if strings.Contains(output.String(), "aotProtocolFn") {
		t.Fatalf("protocol direct linking ignored ^:redef:\n%s", output.String())
	}
}

func TestGenerateStableMixedNumericLoop(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.mixed-numeric-loop"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn float-loop []
		  (loop [i 0
		         value 0.25
		         checksum 0.0]
		    (if (= i 100)
		      [value checksum]
		      (recur (inc i)
		             (* value 1.0000001)
		             (+ checksum value)))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate mixed numeric loop: %v", err)
	}
	generated := output.String()
	if got := strings.Count(generated, " float64 = "); got < 4 {
		t.Fatalf("mixed numeric loop retained boxed float state:\n%s", generated)
	}
	for _, boxed := range []string{
		"lang.Numbers.Multiply",
		"lang.Numbers.Add",
	} {
		if strings.Contains(generated, boxed) {
			t.Fatalf("mixed numeric loop retained %s:\n%s", boxed, generated)
		}
	}
}

func TestGenerateTypedCoreModulus(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.typed-modulus"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn modulus-loop []
		  (loop [i 0
		         total 0]
		    (if (= i 100)
		      [i total]
		      (recur (inc i) (+ total (mod i 7))))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate typed modulus: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "lang.ModInt64(") {
		t.Fatalf("typed modulus retained generic Var dispatch:\n%s", generated)
	}
}

func TestGenerateStaticInstanceCheck(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.static-instance"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn vector-value? [x]
		  (instance? github.com:glojurelang:glojure:pkg:lang.IPersistentVector x))
		(defn dynamic-instance? [t x]
		  (instance? t x))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate static instance check: %v", err)
	}
	generated := output.String()
	const directCheck = "lang.IsInstance[lang.IPersistentVector]"
	if !strings.Contains(generated, directCheck) {
		t.Fatalf("known instance? target did not use a type assertion:\n%s", generated)
	}
	if got := strings.Count(generated, directCheck); got != 1 {
		t.Fatalf("generated %d static vector checks, want 1:\n%s", got, generated)
	}
}

func TestGenerateStaticInstanceCheckForJVMClass(t *testing.T) {
	instanceVar := lang.NSCore.FindInternedVar(lang.NewSymbol("instance?"))
	if instanceVar == nil {
		t.Fatal("clojure.core/instance? is not interned")
	}

	var output bytes.Buffer
	generator := newGenerator(&output, true)
	generator.addImport("github.com/glojurelang/glojure/pkg/lang")
	generator.currentWriter = &output
	invoke := &ast.InvokeNode{
		Fn: &ast.Node{
			Op:  ast.OpVar,
			Sub: &ast.VarNode{Var: instanceVar},
		},
		Args: []*ast.Node{
			{
				Op: ast.OpConst,
				Sub: &ast.ConstNode{
					Value: lang.NewClass(
						reflect.TypeOf(false),
						"java.lang.Boolean",
					),
				},
			},
			{Op: ast.OpConst, Sub: &ast.ConstNode{Value: true}},
		},
	}

	call, ok := generator.staticInstanceCall(invoke, []string{"", "value"})
	if !ok {
		t.Fatal("JVM class did not produce a static instance check")
	}
	if call != "lang.IsInstance[bool](value)" {
		t.Fatalf("static instance check = %q, want bool type", call)
	}
	if strings.Contains(output.String(), "java.lang") {
		t.Fatalf("JVM class produced an invalid Go import:\n%s", output.String())
	}
}

func TestGenerateStaticInstanceCheckForSameNamespaceDirectLink(t *testing.T) {
	instanceVar := lang.NSCore.FindInternedVar(lang.NewSymbol("instance?"))
	if instanceVar == nil {
		t.Fatal("clojure.core/instance? is not interned")
	}

	var output bytes.Buffer
	generator := newGenerator(&output, true)
	generator.addImport("github.com/glojurelang/glojure/pkg/lang")
	generator.currentWriter = &output
	generator.aotNamespace = lang.NSCore
	target := &aotSpecializationTarget{
		vr:           instanceVar,
		directLinked: true,
		directFnVar:  "aotDirectFn0",
	}
	target.directArities[2] = true
	generator.aotCallTargets[instanceVar] = target

	vectorType := reflect.TypeOf((*lang.IPersistentVector)(nil)).Elem()
	invoke := &ast.InvokeNode{
		Fn: &ast.Node{
			Op:  ast.OpVar,
			Sub: &ast.VarNode{Var: instanceVar},
		},
		Args: []*ast.Node{
			{Op: ast.OpConst, Sub: &ast.ConstNode{Value: vectorType}},
			{Op: ast.OpConst, Sub: &ast.ConstNode{Value: nil}},
		},
	}
	generator.generateInvokeDefault(invoke)

	generated := output.String()
	if !strings.Contains(
		generated,
		"lang.IsInstance[lang.IPersistentVector](nil)",
	) {
		t.Fatalf("same-namespace instance? did not use a type assertion:\n%s",
			generated)
	}
	if strings.Contains(generated, "aotDirectFn0") {
		t.Fatalf("same-namespace instance? retained function dispatch:\n%s",
			generated)
	}
}

func TestGenerateDirectCallsForKnownFunctionArities(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.direct-known-arities"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn fixed
		  [a b c d e f g h i j k l m n o p q r s t]
		  t)
		(defn choose
		  ([x] x)
		  ([x y] y))
		(defn flexible
		  ([x] x)
		  ([x y & more] more))
		(defn call-fixed []
		  (fixed 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20))
		(defn call-one [] (choose 1))
		(defn call-two [] (choose 1 2))
		(defn call-variadic [] (flexible 1 2 3))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate known function calls: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "lang.FnFunc20") ||
		!strings.Contains(generated, "aotDirectFn") {
		t.Fatalf("known arity-20 function did not receive a direct slot:\n%s", generated)
	}
	if got := strings.Count(generated, " lang.ArityFn\n"); got < 2 {
		t.Fatalf("multi-arity functions received %d direct slots, want at least 2:\n%s",
			got, generated)
	}
	for _, declaration := range []string{
		"Arity1 lang.FnFunc1",
		"Arity2 lang.FnFunc2",
	} {
		if !strings.Contains(generated, declaration) {
			t.Fatalf("known overload omitted typed slot %q:\n%s",
				declaration, generated)
		}
	}
	for _, call := range []string{
		"Arity1(int64(1))",
		"Arity2(int64(1), int64(2))",
		".Invoke3(",
	} {
		if !strings.Contains(generated, call) {
			t.Fatalf("known function call omitted direct %s dispatch:\n%s",
				call, generated)
		}
	}
	if strings.Contains(generated, "RootVersion() ==") {
		t.Fatalf("default direct linking retained a same-namespace Var guard:\n%s",
			generated)
	}
}

func TestInferredDirectLinkingCanBeDisabled(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.guarded-inferred-call"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn target [x] x)
		(defn caller [x] (target x))`)

	var output bytes.Buffer
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate guarded inferred call: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "RootVersion() ==") {
		t.Fatalf("disabled direct linking omitted same-namespace Var guard:\n%s",
			generated)
	}
}

func TestGenerateDirectLinkedRecursiveInt64Function(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.direct-recursive-int64"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn fib [n]
		  (if (<= n 1)
		    n
		    (+ (fib (- n 1)) (fib (- n 2)))))
		(defn run [] (fib 10))`)

	t.Run("default direct link", func(t *testing.T) {
		var output bytes.Buffer
		if err := NewGenerator(&output).Generate(ns); err != nil {
			t.Fatalf("generate direct-linked recursive function: %v", err)
		}
		if generated := output.String(); strings.Contains(
			generated,
			"RootVersion() !=",
		) {
			t.Fatalf(
				"direct-linked recursive specialization retained a Var guard:\n%s",
				generated,
			)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		var output bytes.Buffer
		if err := newGenerator(&output, false).Generate(ns); err != nil {
			t.Fatalf("generate guarded recursive function: %v", err)
		}
		if generated := output.String(); !strings.Contains(
			generated,
			"RootVersion() !=",
		) {
			t.Fatalf(
				"guarded recursive specialization omitted its Var guard:\n%s",
				generated,
			)
		}
	})
}

func TestGenerateDirectLinkedFloat64Callee(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.direct-float64-callee"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn polynomial [x]
		  (+ (* x x) 1.0))
		(defn caller [x]
		  (polynomial x))`)

	t.Run("default direct link", func(t *testing.T) {
		var output bytes.Buffer
		if err := NewGenerator(&output).Generate(ns); err != nil {
			t.Fatalf("generate direct-linked float64 callee: %v", err)
		}
		if generated := output.String(); strings.Contains(
			generated,
			"RootVersion() !=",
		) {
			t.Fatalf(
				"direct-linked float64 specialization retained a Var guard:\n%s",
				generated,
			)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		var output bytes.Buffer
		if err := newGenerator(&output, false).Generate(ns); err != nil {
			t.Fatalf("generate guarded float64 callee: %v", err)
		}
		if generated := output.String(); !strings.Contains(
			generated,
			"RootVersion() !=",
		) {
			t.Fatalf(
				"guarded float64 specialization omitted its Var guard:\n%s",
				generated,
			)
		}
	})
}

func TestGenerateSharedStaticKeywordMapShapes(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.static-keyword-maps"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn state-one [value]
		  {:a value, :b 2, :c 3, :d 4, :e 5, :f 6, :g 7, :h 8, :i 9})
		(defn state-two [value]
		  {:a value, :b 2, :c 3, :d 4, :e 5, :f 6, :g 7, :h 8, :i 9})`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate static keyword maps: %v", err)
	}
	generated := output.String()
	if got := strings.Count(generated, "lang.NewKeywordMapShape("); got != 1 {
		t.Fatalf("generated %d keyword map shapes, want 1:\n%s", got, generated)
	}
	if got := strings.Count(generated, "aotKeywordMapNew0("); got != 3 {
		t.Fatalf("generated keyword map constructor occurred %d times, want 3:\n%s",
			got, generated)
	}
	if got := strings.Count(generated, "aotKeywordMapStorage0 struct"); got != 1 {
		t.Fatalf("generated %d keyword map storage types, want 1:\n%s", got, generated)
	}
	if strings.Contains(generated, "lang.NewStaticKeywordMap(") {
		t.Fatalf("generated static keyword maps retained variadic storage:\n%s", generated)
	}
}

func TestGenerateStaticKeywordMapMetadataUsesInterfaceAssertion(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.static-keyword-map-meta"))
	var output bytes.Buffer
	generator := NewGenerator(&output)
	constant := func(value any) *ast.Node {
		node := ast.MakeNode(ast.OpConst, value)
		node.Sub = &ast.ConstNode{Value: value}
		return node
	}
	meta := ast.MakeNode(ast.OpMap, nil)
	meta.Sub = &ast.MapNode{
		Keys: []*ast.Node{
			constant(lang.NewKeyword("a")),
			constant(lang.NewKeyword("b")),
			constant(lang.NewKeyword("c")),
		},
		Vals: []*ast.Node{
			constant(int64(1)),
			constant(int64(2)),
			constant(int64(3)),
		},
	}
	name := lang.NewSymbol("value")
	definition := ast.MakeNode(ast.OpDef, nil)
	definition.Sub = &ast.DefNode{
		Name: name,
		Var:  ns.Intern(name),
		Meta: meta,
	}
	generator.currentIR = compiler.BuildTypedIR(definition)
	generator.generateDef(definition)

	generated := output.String()
	if !strings.Contains(generated, ".SetMeta(any(") {
		t.Fatalf("generated metadata did not normalize the concrete map to an interface:\n%s",
			generated)
	}
}

func TestGenerateTypedIRFixedGetIn(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.typed-ir-get-in"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn callback [receiver name]
		  (get-in receiver [:callbacks name]))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate fixed get-in: %v", err)
	}
	generated := output.String()
	if got := strings.Count(generated, "lang.Get("); got < 2 {
		t.Fatalf("fixed get-in emitted %d direct lookups, want at least 2:\n%s",
			got, generated)
	}
	if strings.Contains(generated, "lang.NewVector(kw_callbacks") {
		t.Fatalf("fixed get-in retained its path vector:\n%s", generated)
	}
}

func TestGenerateTypedIRFixedGetInNeedsDirectLinking(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-get-in-guarded"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn callback [receiver name]
		  (get-in receiver [:callbacks name]))`)

	var output bytes.Buffer
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate guarded fixed get-in: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "lang.NewVector(kw_callbacks") {
		t.Fatalf("disabled direct linking incorrectly fused get-in:\n%s", generated)
	}
}

func TestGenerateTypedIRFixedGetInEvaluatesAllKeysBeforeLookup(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-get-in-order"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn callback [receiver]
		  (get-in receiver [[1] [2]]))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate ordered fixed get-in: %v", err)
	}
	generated := output.String()
	firstKey := strings.Index(generated, "lang.NewVector(int64(1))")
	secondKey := strings.Index(generated, "lang.NewVector(int64(2))")
	firstLookup := strings.Index(generated, "lang.Get(")
	if firstKey < 0 || secondKey < 0 || firstLookup < 0 ||
		firstLookup < firstKey || firstLookup < secondKey {
		t.Fatalf("lookup ran before every path expression was evaluated:\n%s",
			generated)
	}
}

func TestGenerateTypedIRSmallKeywordMapShape(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-small-keyword-map"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn result [a b c]
		  {:try a, :got b, :not c})`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate small keyword map: %v", err)
	}
	generated := output.String()
	if !strings.Contains(
		generated,
		`lang.NewKeywordMapShape("try", "got", "not")`,
	) {
		t.Fatalf("small keyword map did not receive a fixed shape:\n%s", generated)
	}
	if !strings.Contains(generated, "aotKeywordMapStorage0 struct") {
		t.Fatalf("small keyword map omitted co-allocated storage:\n%s", generated)
	}
}

func TestGenerateTypedIRNonEscapingSwapCallback(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-direct-swap"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn update-state [state captured]
		  (swap! state (fn [old] [old captured])))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate non-escaping swap callback: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, ".SwapFunc(func(") {
		t.Fatalf("swap! callback retained its IFn wrapper:\n%s", generated)
	}
	if !strings.Contains(generated, "runtime.DirectSwap0(") {
		t.Fatalf("swap! callback omitted the generic IAtom fallback:\n%s",
			generated)
	}
}

func TestGenerateTypedIRConfinedStringStack(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-string-stack"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(require '[clojure.string :as str])
		(defn joined []
		  (let [tail (str "tail")
		        parts (atom [])]
		    (swap! parts #(cons tail %))
		    (str/join "__" (cons "head" @parts))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate confined string stack: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"[]string",
		" = append(",
		".Join(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("string-stack fusion omitted %q:\n%s", expected, generated)
		}
	}
	if strings.Contains(generated, "lang.NewCons(") {
		t.Fatalf("string-stack fusion retained cons allocation:\n%s", generated)
	}
}

func TestGenerateTypedIRDoesNotFuseObservableStringStackResult(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-observable-string-stack"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(require '[clojure.string :as str])
		(defn joined []
		  (let [parts (atom [])]
		    (swap! parts #(cons "tail" %))
		    [(swap! parts #(cons "visible" %))
		     (str/join "__" (cons "head" @parts))]))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate observable string stack: %v", err)
	}
	if strings.Contains(output.String(), ".Join(") {
		t.Fatalf("observable swap! result was incorrectly fused:\n%s",
			output.String())
	}
}

func TestGenerateTypedIRDoesNotFuseCapturedStringStack(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-captured-string-stack"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(require '[clojure.string :as str])
		(defn joined []
		  (let [parts (atom [])]
		    (swap! parts #(cons "tail" %))
		    ((fn [] @parts))
		    (str/join "__" (cons "head" @parts))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate captured string stack: %v", err)
	}
	if strings.Contains(output.String(), ".Join(") {
		t.Fatalf("captured atom was incorrectly fused:\n%s",
			output.String())
	}
}

func TestGenerateTypedIRDoesNotFuseNonStringStackValues(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-ir-dynamic-string-stack"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(require '[clojure.string :as str])
		(defn joined [value]
		  (let [parts (atom [])]
		    (swap! parts #(cons value %))
		    (str/join "__" (cons "head" @parts))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate dynamic string stack: %v", err)
	}
	if strings.Contains(output.String(), ".Join(") {
		t.Fatalf("dynamic stack value was incorrectly coerced early:\n%s",
			output.String())
	}
}

func TestGenerateOwnedLoopStringParts(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.owned-string-parts"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn concatenate [values]
		  (loop [remaining (seq values)
		         parts []]
		    (if remaining
		      (recur (next remaining)
		             (conj parts (first remaining)))
		      (apply str parts))))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate owned string parts: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"[]any",
		" = append(",
		"runtime.ConcatStringParts(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("owned string-parts lowering omitted %q:\n%s",
				expected, generated)
		}
	}

	var fallback bytes.Buffer
	if err := newGenerator(&fallback, false).Generate(ns); err != nil {
		t.Fatalf("generate string parts without direct linking: %v", err)
	}
	if strings.Contains(fallback.String(), "runtime.ConcatStringParts(") {
		t.Fatalf(
			"disabled direct linking retained owned string parts:\n%s",
			fallback.String(),
		)
	}
}

func TestGenerateUniformIntegerCaseWithPrimitiveResult(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.primitive-case-result"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn weighted-sum [values]
		  (loop [remaining (seq values)
		         total 0]
		    (if remaining
		      (recur (next remaining)
		             (+ total
		                (case (first remaining)
		                  \A 1
		                  \C 2
		                  \G 3
		                  \T 4)))
		      total)))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate primitive case result: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "CheckedAddInt64(") {
		t.Fatalf("uniform integer case did not preserve primitive arithmetic:\n%s",
			generated)
	}
	if strings.Contains(generated, "lang.Numbers.Add(") {
		t.Fatalf("uniform integer case retained boxed arithmetic:\n%s",
			generated)
	}
}

func TestGenerateGuardedInt64ParameterRegion(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.guarded-int64-parameters"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn next-state [state]
		  (mod (+ (* state 3) 1) 97))
		(defn wrapped [state]
		  {:state (next-state state)
		   :label "ok"})
		(defn ^:redef redef-next-state [state]
		  (mod (+ (* state 5) 1) 101))
		(defn wrapped-redef [state]
		  {:state (redef-next-state state)})
		(defn large-leaf [state]
		  (let [s1 (inc state)
		        s2 (inc s1)
		        s3 (inc s2)
		        s4 (inc s3)
		        s5 (inc s4)
		        s6 (inc s5)
		        s7 (inc s6)
		        s8 (inc s7)
		        s9 (inc s8)
		        s10 (inc s9)
		        s11 (inc s10)
		        s12 (inc s11)]
		    s12))
		(defn wrapped-large [state]
		  {:state (large-leaf state)})`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate guarded int64 parameters: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "// inline int64 call") ||
		strings.Count(generated, "lang.ModInt64(") < 2 {
		t.Fatalf("guarded int64 region did not inline the typed leaf:\n%s",
			generated)
	}
	if !strings.Contains(generated, "lang.CheckedMultiplyInt64(") {
		t.Fatalf("inlined primitive leaf lost checked overflow semantics:\n%s",
			generated)
	}
	for _, name := range []string{"redef-next-state", "large-leaf"} {
		if strings.Contains(
			generated,
			"// inline int64 call #'"+
				"codegen.guarded-int64-parameters/"+name,
		) {
			t.Fatalf("ineligible %s call was inlined:\n%s", name, generated)
		}
	}

	var fallback bytes.Buffer
	if err := newGenerator(&fallback, false).Generate(ns); err != nil {
		t.Fatalf("generate without direct linking: %v", err)
	}
	if strings.Contains(fallback.String(), "// inline int64 call") {
		t.Fatalf("disabled direct linking retained int64 inlining:\n%s",
			fallback.String())
	}
}

func TestGenerateTypedInt64ParameterDirectCallWithDynamicResult(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.typed-int64-parameter-call"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn make-event [index]
		  {:value (mod (* index 37) 1000)})
		(defn run-one []
		  (make-event 7))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate typed parameter call: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"var aotInt64ParamFn",
		"// direct int64 parameter call",
		"lang.CheckedMultiplyInt64(",
		"lang.ModInt64(",
		".(int64)",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("typed parameter call omitted %q:\n%s",
				expected, generated)
		}
	}

	var fallback bytes.Buffer
	if err := newGenerator(&fallback, false).Generate(ns); err != nil {
		t.Fatalf("generate typed call without direct linking: %v", err)
	}
	if strings.Contains(fallback.String(), "aotInt64ParamFn") {
		t.Fatalf("disabled direct linking retained typed parameter call:\n%s",
			fallback.String())
	}
}

func TestGenerateStableCompositeMultiFnDispatch(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.stable-composite-multifn"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defmulti choose
		  (fn [event] [(:kind event) (:priority event)]))
		(defmethod choose [:read :low] [event]
		  (:value event))
		(defmethod choose :default [_]
		  0)
		(defn run-one []
		  (choose {:kind :read :priority :low :value 42}))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate composite multimethod: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"var aotMultiFnFast",
		"var aotMultiFnDispatch",
		".IsGeneration(",
		"// scalar multimethod dispatch",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("scalar multimethod dispatch omitted %q:\n%s",
				expected, generated)
		}
	}

	var fallback bytes.Buffer
	if err := newGenerator(&fallback, false).Generate(ns); err != nil {
		t.Fatalf("generate multimethod without direct linking: %v", err)
	}
	if strings.Contains(fallback.String(), "aotMultiFnFast") {
		t.Fatalf("disabled direct linking retained multimethod fast path:\n%s",
			fallback.String())
	}
}

func TestGenerateSeqTruthinessWithoutSequenceAllocation(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.seq-truthiness"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn has-items? [xs]
		  (if (seq xs) true false))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate seq truthiness: %v", err)
	}
	if !strings.Contains(output.String(), "lang.IsSeqTruthy(") {
		t.Fatalf("generated code did not specialize seq truthiness:\n%s", output.String())
	}
	if strings.Contains(output.String(), "aotExternalDefault") {
		t.Fatalf("default core direct linking retained a Var guard:\n%s", output.String())
	}
}

func TestCoreDirectLinkingCanBeDisabled(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.guarded-core-call"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn call-identity [x]
		  (identity x))`)

	for _, test := range []struct {
		name       string
		directLink bool
		want       string
		notWant    string
	}{
		{
			name:       "default direct link",
			directLink: true,
			want:       "aotLinkFn1",
			notWant:    "aotCacheFn1",
		},
		{
			name:       "disabled",
			directLink: false,
			want:       "aotCacheFn1",
			notWant:    "aotLinkFn1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := newGenerator(&output, test.directLink).Generate(ns); err != nil {
				t.Fatalf("generate core call: %v", err)
			}
			generated := output.String()
			if !strings.Contains(generated, test.want) {
				t.Fatalf("generated core call omitted %q:\n%s", test.want, generated)
			}
			if strings.Contains(generated, test.notWant) {
				t.Fatalf("generated core call unexpectedly retained %q:\n%s",
					test.notWant, generated)
			}
			if test.directLink {
				for _, bootstrapGuard := range []string{
					"if vr.IsBound()",
					"var once sync.Once",
				} {
					if !strings.Contains(generated, bootstrapGuard) {
						t.Fatalf("direct-link adapter omitted bootstrap guard %q:\n%s",
							bootstrapGuard, generated)
					}
				}
			}
		})
	}
}

func TestDirectLinkingUsesCompilerOptions(t *testing.T) {
	compilerOptions := lang.NSCore.FindInternedVar(
		lang.NewSymbol("*compiler-options*"),
	)
	if compilerOptions == nil {
		t.Fatal("clojure.core/*compiler-options* is not interned")
	}

	t.Run("explicit false", func(t *testing.T) {
		lang.PushThreadBindings(lang.NewMap(
			compilerOptions,
			lang.NewMap(lang.KWDirectLinking, false),
		))
		defer lang.PopThreadBindings()
		if aotDirectLinkEnabled() {
			t.Fatal("{:direct-linking false} left direct linking enabled")
		}
	})

	t.Run("absent", func(t *testing.T) {
		lang.PushThreadBindings(lang.NewMap(
			compilerOptions,
			lang.NewMap(),
		))
		defer lang.PopThreadBindings()
		if !aotDirectLinkEnabled() {
			t.Fatal("direct linking is not enabled when the option is absent")
		}
	})
}

func TestGenerateOwnedMapReduceUsesSharedIRWithDirectLinking(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.owned-map-reduce"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn summarize [values]
		  (reduce
		    (fn [totals value]
		      (-> totals
		          (update-in [(:service value) :count] (fnil inc 0))
		          (update-in [(:service value) :sum] (fnil + 0) (:n value))))
		    {}
		    values))`)

	var output bytes.Buffer
	if err := newGenerator(&output, true).Generate(ns); err != nil {
		t.Fatalf("generate owned map reduce: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		"runtime.ReduceOwnedMap(",
		"runtime.UpdateOwnedMapPath2Default3(",
		"runtime.UpdateOwnedMapPath2Default4(",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("owned map reduce omitted %q:\n%s", expected, generated)
		}
	}

	func() {
		fnil := lang.NSCore.FindInternedVar(lang.NewSymbol("fnil"))
		originalFnilMeta := fnil.Meta()
		fnil.SetMeta(
			originalFnilMeta.Assoc(
				lang.KWRedef,
				true,
			).(lang.IPersistentMap),
		)
		defer fnil.SetMeta(originalFnilMeta)

		output.Reset()
		if err := newGenerator(&output, true).Generate(ns); err != nil {
			t.Fatalf("generate redefinable fnil owned map reduce: %v", err)
		}
		generated = output.String()
		if !strings.Contains(generated, "runtime.ReduceOwnedMap(") ||
			!strings.Contains(generated, "runtime.UpdateOwnedMapPath2_3(") ||
			strings.Contains(generated, "runtime.UpdateOwnedMapPath2Default") {
			t.Fatalf(
				"redefinable fnil did not retain the wrapper fallback:\n%s",
				generated,
			)
		}
	}()

	output.Reset()
	if err := newGenerator(&output, false).Generate(ns); err != nil {
		t.Fatalf("generate non-direct owned map reduce: %v", err)
	}
	generated = output.String()
	if strings.Contains(generated, "runtime.ReduceOwnedMap(") ||
		strings.Contains(generated, "runtime.UpdateOwnedMap") {
		t.Fatalf("owned map reduction ignored disabled direct linking:\n%s", generated)
	}

	updateIn := lang.NSCore.FindInternedVar(lang.NewSymbol("update-in"))
	originalMeta := updateIn.Meta()
	updateIn.SetMeta(originalMeta.Assoc(lang.KWRedef, true).(lang.IPersistentMap))
	defer updateIn.SetMeta(originalMeta)
	output.Reset()
	if err := newGenerator(&output, true).Generate(ns); err != nil {
		t.Fatalf("generate redefinable owned map reduce: %v", err)
	}
	if generated = output.String(); strings.Contains(
		generated,
		"runtime.ReduceOwnedMap(",
	) {
		t.Fatalf("owned map reduction bypassed ^:redef update-in:\n%s", generated)
	}
}

func TestCoreDirectLinkingHonorsRedefMetadata(t *testing.T) {
	identity := lang.NSCore.FindInternedVar(lang.NewSymbol("identity"))
	originalMeta := identity.Meta()
	identity.SetMeta(originalMeta.Assoc(lang.KWRedef, true).(lang.IPersistentMap))
	defer identity.SetMeta(originalMeta)

	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.redef-core-call"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn call-redef-identity [x]
		  (identity x))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate ^:redef core call: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "aotCacheFn1") {
		t.Fatalf("^:redef core call was linked directly:\n%s", generated)
	}
	if strings.Contains(generated, "aotLinkFn1") {
		t.Fatalf("^:redef core call received a direct-link adapter:\n%s", generated)
	}
}

func TestInferredDirectLinkingHonorsRedefMetadata(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.redef-inferred-call"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn ^:redef target [x] x)
		(defn caller [x] (target x))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate ^:redef inferred call: %v", err)
	}
	if generated := output.String(); !strings.Contains(generated, "RootVersion() ==") {
		t.Fatalf("^:redef inferred call omitted its Var guard:\n%s", generated)
	}
}

func TestGeneratedFunctionCachesResolvedIntegerReturn(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.boxed-host-return"))
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn compare-result [x y]
		  (github.com:glojurelang:glojure:pkg:lang.Compare x y))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate resolved integer return: %v", err)
	}
	generated := output.String()
	if !strings.Contains(generated, "return lang.BoxInt(") {
		t.Fatalf("resolved Go int return was not boxed through the cache:\n%s", generated)
	}
}

type codegenNamedInt int64

func TestGeneratedFunctionPreservesNamedIntegerReturn(t *testing.T) {
	call := aotTestConst(func(any) codegenNamedInt { return 1 })
	invoke := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   call,
			Args: []*ast.Node{aotTestConst(int64(42))},
		},
	}
	generator := NewGenerator(&bytes.Buffer{})
	generator.currentIR = compiler.BuildTypedIR(invoke)
	if got := generator.boxDynamicResult(invoke, "value"); got != "value" {
		t.Fatalf("named integer return was coerced at dynamic boundary: %s", got)
	}
}

func TestGenerateMutableRuntimeValues(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})

	if got, want := generator.generateValue(lang.NewVolatile(int64(7))),
		"lang.NewVolatile(int64(7))"; got != want {
		t.Fatalf("generated volatile = %q, want %q", got, want)
	}

	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.mutable-values"))
	delay := lang.NewDelay(ns.Intern(lang.NewSymbol("delayed")))
	got := generator.generateValue(delay)
	if !strings.HasPrefix(got, "lang.NewDelay(") {
		t.Fatalf("generated delay = %q, want lang.NewDelay expression", got)
	}
}

func TestGenerateResolvedHostReference(t *testing.T) {
	generator := NewGenerator(&bytes.Buffer{})
	node := ast.MakeNode(ast.OpConst, nil)
	node.Sub = &ast.ConstNode{
		Value:      new(int),
		HostSymbol: lang.NewSymbol("example.com:host.Value"),
	}

	if got, want := generator.generateASTNode(node), "host0.Value"; got != want {
		t.Fatalf("generated host reference = %q, want %q", got, want)
	}
	if got, want := generator.imports["example.com/host"], "host0"; got != want {
		t.Fatalf("host import alias = %q, want %q", got, want)
	}
}

func TestGenerateLateBoundHostForm(t *testing.T) {
	var output bytes.Buffer
	generator := NewGenerator(&output)
	node := ast.MakeNode(ast.OpMaybeHostForm, nil)
	node.Sub = &ast.MaybeHostFormNode{
		Class: "clojure.lang.MapEntry",
		Field: lang.NewSymbol("create"),
	}

	result := generator.generateASTNode(node)
	if result == "nil" {
		t.Fatal("late-bound host form was discarded")
	}
	if got := output.String(); !strings.Contains(
		got,
		`Get("clojure.lang.MapEntry.create")`,
	) {
		t.Fatalf("generated host lookup = %q", got)
	}
}

func TestGenerateResolvedHostClassValue(t *testing.T) {
	var output bytes.Buffer
	generator := NewGenerator(&output)
	node := ast.MakeNode(ast.OpConst, nil)
	node.Sub = &ast.ConstNode{
		Value:      lang.NewClass(reflect.TypeOf(int64(0)), "java.lang.Long"),
		HostSymbol: lang.NewSymbol("java.lang.Long"),
	}

	if got := generator.generateASTNode(node); got == "nil" {
		t.Fatal("resolved host class generated nil")
	}
	if got := output.String(); !strings.Contains(got, `"java.lang.Long"`) {
		t.Fatalf("generated host class did not retain Java identity: %s", got)
	}
	if _, ok := generator.imports["java.lang"]; ok {
		t.Fatal("resolved host class generated a java.lang import")
	}
}

func TestGenerateKeywordInvocationUsesDirectFixedArityCall(t *testing.T) {
	var output bytes.Buffer
	generator := NewGenerator(&output)
	keyword := aotTestConst(lang.NewKeyword("answer"))
	invoke := ast.MakeNode(ast.OpInvoke, nil)
	invoke.Sub = &ast.InvokeNode{
		Fn:   keyword,
		Args: []*ast.Node{aotTestConst(nil), aotTestConst(int64(42))},
	}

	result := generator.generateASTNode(invoke)
	generated := output.String()
	if !strings.Contains(generated, ".Invoke2(nil, int64(42))") {
		t.Fatalf("keyword invocation was not emitted directly: %s = %s", result, generated)
	}
	if strings.Contains(generated, "lang.Apply2") {
		t.Fatalf("keyword invocation retained generic apply dispatch: %s", generated)
	}
}

func TestDirectHostMethod(t *testing.T) {
	target := &ast.Node{
		Op: ast.OpConst,
		Sub: &ast.ConstNode{
			Value: lang.Numbers,
		},
	}

	if got, ok := directHostMethod(target, "multiply", 2); !ok || got != "Multiply" {
		t.Fatalf("directHostMethod(multiply) = %q, %v", got, ok)
	}
	if got, ok := directHostMethod(target, "UncheckedIntDivide", 2); ok {
		t.Fatalf("typed method unexpectedly resolved directly as %q", got)
	}
	if got, ok := directHostMethod(target, "multiply", 1); ok {
		t.Fatalf("wrong arity unexpectedly resolved directly as %q", got)
	}
}

func TestDirectHostCallConvertsIntegerArguments(t *testing.T) {
	target := &ast.Node{
		Op: ast.OpConst,
		Sub: &ast.ConstNode{
			Value: RT,
		},
	}

	method, args, ok := directHostCall(
		target,
		"Nth",
		[]string{"collection", "index"},
	)
	if !ok || method != "Nth" {
		t.Fatalf("directHostCall(Nth) = %q, %v, %v", method, args, ok)
	}
	if got, want := args[0], "collection"; got != want {
		t.Fatalf("collection argument = %q, want %q", got, want)
	}
	if got, want := args[1], "lang.IntCast(index)"; got != want {
		t.Fatalf("index argument = %q, want %q", got, want)
	}

	for _, args := range [][]string{
		{"collection", "key"},
		{"collection", "key", "notFound"},
	} {
		method, converted, ok := directHostCall(target, "Get", args)
		if !ok || method != "Get" {
			t.Fatalf("directHostCall(Get, %d args) = %q, %v, %v", len(args), method, converted, ok)
		}
		if !reflect.DeepEqual(converted, args) {
			t.Fatalf("directHostCall(Get) args = %v, want %v", converted, args)
		}
	}

	if method, _, ok := directHostCall(target, "Get", []string{"collection"}); ok {
		t.Fatalf("undersupplied variadic call unexpectedly resolved directly as %q", method)
	}
}

func TestDirectTaggedHostCallUsesInferredMethodSet(t *testing.T) {
	const volatileType = "github.com:glojurelang:glojure:pkg:lang.Volatile"
	pkgmap.Set(
		"github.com/glojurelang/glojure/pkg/lang.Volatile",
		reflect.TypeOf(lang.Volatile{}),
	)
	taggedName := lang.NewSymbol("value").WithMeta(
		lang.NewMap(lang.KWTag, lang.NewSymbol(volatileType)),
	).(*lang.Symbol)
	form := lang.NewSymbol("value").WithMeta(
		lang.NewMap(lang.KWLine, int64(1)),
	)
	target := ast.MakeNode(ast.OpLocal, form)
	target.Sub = &ast.LocalNode{Name: taggedName}

	generator := NewGenerator(&bytes.Buffer{})
	method, receiver, args, ok := generator.directInferredHostCall(
		target,
		"value",
		"reset",
		[]string{"replacement"},
	)
	if !ok {
		t.Fatal("tagged Volatile Reset call was not resolved directly")
	}
	if want := "Reset"; method != want {
		t.Fatalf("method = %q, want %q", method, want)
	}
	if want := "value.(interface { Reset(any) any })"; receiver != want {
		t.Fatalf("receiver = %q, want %q", receiver, want)
	}
	if want := []string{"replacement"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestDirectKnownHostCallConvertsInterfaceArguments(t *testing.T) {
	target := ast.MakeNode(ast.OpConst, nil)
	target.Sub = &ast.ConstNode{Value: RT}

	generator := NewGenerator(&bytes.Buffer{})
	generator.addImport("github.com/glojurelang/glojure/pkg/lang")
	generator.addImport("github.com/glojurelang/glojure/pkg/runtime")
	method, receiver, args, ok := generator.directInferredHostCall(
		target,
		"runtime.RT",
		"Subvec",
		[]string{"vector", "start", "end"},
	)
	if !ok {
		t.Fatal("known RT.Subvec call was not resolved directly")
	}
	if want := "Subvec"; method != want {
		t.Fatalf("method = %q, want %q", method, want)
	}
	if want := "runtime.RT"; receiver != want {
		t.Fatalf("receiver = %q, want %q", receiver, want)
	}
	wantArgs := []string{
		"lang.MustHostCast[lang.IPersistentVector](vector)",
		"start",
		"end",
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
}

func TestDirectTaggedHostCallUsesFormMetadata(t *testing.T) {
	const derefType = "github.com:glojurelang:glojure:pkg:lang.IDeref"
	pkgmap.Set(
		"github.com/glojurelang/glojure/pkg/lang.IDeref",
		reflect.TypeFor[lang.IDeref](),
	)
	form := lang.NewSymbol("value").WithMeta(
		lang.NewMap(lang.KWTag, lang.NewSymbol(derefType)),
	).(*lang.Symbol)
	target := ast.MakeNode(ast.OpLocal, form)
	target.Sub = &ast.LocalNode{Name: lang.NewSymbol("value")}

	generator := NewGenerator(&bytes.Buffer{})
	method, receiver, args, ok := generator.directInferredHostCall(
		target,
		"value",
		"deref",
		nil,
	)
	if !ok {
		t.Fatal("tagged IDeref call was not resolved directly")
	}
	if want := "Deref"; method != want {
		t.Fatalf("method = %q, want %q", method, want)
	}
	if want := "value.(interface { Deref() any })"; receiver != want {
		t.Fatalf("receiver = %q, want %q", receiver, want)
	}
	if len(args) != 0 {
		t.Fatalf("args = %v, want none", args)
	}
}

func TestDirectTaggedHostCallResolvesClojureLangAlias(t *testing.T) {
	pkgmap.Set(
		"github.com/glojurelang/glojure/pkg/lang.IChunk",
		reflect.TypeFor[lang.IChunk](),
	)
	taggedName := lang.NewSymbol("chunk").WithMeta(
		lang.NewMap(lang.KWTag, lang.NewSymbol("clojure.lang.IChunk")),
	).(*lang.Symbol)
	target := ast.MakeNode(ast.OpLocal, taggedName)
	target.Sub = &ast.LocalNode{Name: taggedName}

	generator := NewGenerator(&bytes.Buffer{})
	method, receiver, args, ok := generator.directInferredHostCall(
		target,
		"chunk",
		"nth",
		[]string{"index"},
	)
	if !ok {
		t.Fatal("clojure.lang.IChunk Nth call was not resolved directly")
	}
	if want := "Nth"; method != want {
		t.Fatalf("method = %q, want %q", method, want)
	}
	if want := "chunk.(interface { Nth(int) any })"; receiver != want {
		t.Fatalf("receiver = %q, want %q", receiver, want)
	}
	if want := []string{"lang.IntCast(index)"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestDirectTaggedHostCallRequiresResolvableType(t *testing.T) {
	target := ast.MakeNode(ast.OpLocal, lang.NewSymbol("value"))
	target.Sub = &ast.LocalNode{Name: lang.NewSymbol("value")}

	generator := NewGenerator(&bytes.Buffer{})
	if method, _, _, ok := generator.directInferredHostCall(
		target,
		"value",
		"deref",
		nil,
	); ok {
		t.Fatalf("untagged call unexpectedly resolved directly as %q", method)
	}
}

func TestLoadedNamespacesUseFreshRuntimeState(t *testing.T) {
	core := lang.FindOrCreateNamespace(lang.NewSymbol("clojure.core"))
	loadedLibs := core.Intern(lang.NewSymbol("*loaded-libs*"))

	initializer, ok := runtimeStateInitializer(loadedLibs)
	if !ok {
		t.Fatal("*loaded-libs* does not have a runtime-state initializer")
	}
	if want := "lang.NewRef(lang.NewSet())"; initializer != want {
		t.Fatalf("*loaded-libs* initializer = %q, want %q", initializer, want)
	}
}

func TestRuntimeFunctionMeta(t *testing.T) {
	foo := lang.NewKeyword("foo")
	bar := lang.NewKeyword("bar")
	retTag := lang.NewKeyword("rettag")

	if got := runtimeFunctionMeta(nil); got != nil {
		t.Fatalf("nil metadata became %v", got)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil)); got != nil {
		t.Fatalf("compiler-only metadata became %v", got)
	}

	explicit := lang.NewMap(foo, bar)
	if got := runtimeFunctionMeta(explicit); !lang.Equals(got, explicit) {
		t.Fatalf("explicit metadata became %v, want %v", got, explicit)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil, foo, bar)); !lang.Equals(got, explicit) {
		t.Fatalf("mixed metadata became %v, want %v", got, explicit)
	}
}

func TestAnalyzeInt64AOTRecursiveFunction(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-analysis"))
	vr := ns.Intern(lang.NewSymbol("fib"))
	n := lang.NewSymbol("n")
	localN := aotTestLocal(n)

	body := aotTestIf(
		aotTestNumbersCall("Lte", localN, aotTestInt(1)),
		localN,
		aotTestNumbersCall(
			"Add",
			aotTestInvoke(vr, aotTestNumbersCall("Minus", localN, aotTestInt(1))),
			aotTestInvoke(vr, aotTestNumbersCall("Minus", localN, aotTestInt(2))),
		),
	)
	method := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       body,
	}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 recursive function was not specialized")
	}
	if !analysis.usesSelf {
		t.Fatal("recursive function did not request a root-version guard")
	}

	floatBody := aotTestIf(
		aotTestNumbersCall("Lte", localN, aotTestInt(1)),
		aotTestConst(float64(1)),
		localN,
	)
	method.Body = floatBody
	if analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	); analysis != nil {
		t.Fatal("mixed float function unexpectedly received an int64 specialization")
	}
}

func TestAnalyzeInt64AOTLoop(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-loop-analysis"))
	vr := ns.Intern(lang.NewSymbol("sum-loop"))
	i := lang.NewSymbol("i")
	sum := lang.NewSymbol("sum")
	loopID := lang.NewSymbol("loop-id")

	inc := aotTestNumbersCall("Inc", aotTestLocal(i))
	add := aotTestNumbersCall("Add", aotTestLocal(sum), aotTestLocal(i))
	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs: []*ast.Node{
			inc,
			add,
		},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(0)),
			aotTestBinding(sum, aotTestInt(0)),
		},
		Body: aotTestIf(
			aotTestNumbersCall("Lt", aotTestLocal(i), aotTestInt(10)),
			recur,
			aotTestLocal(sum),
		),
	}
	method := &ast.FnMethodNode{Body: loop}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 loop was not specialized")
	}
	if analysis.usesSelf {
		t.Fatal("non-recursive loop unnecessarily requested a root-version guard")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[inc.Sub.(*ast.HostCallNode)] {
		t.Fatal("bounded induction increment retained an overflow check")
	}
	if !analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] {
		t.Fatal("bounded accumulator addition retained an overflow check")
	}
}

func TestInt64AOTRangeProofRetainsPossibleOverflow(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-overflow-proof"))
	vr := ns.Intern(lang.NewSymbol("unsafe-loop"))
	i := lang.NewSymbol("i")
	sum := lang.NewSymbol("sum")
	loopID := lang.NewSymbol("loop-id")

	inc := aotTestNumbersCall("Inc", aotTestLocal(i))
	add := aotTestNumbersCall("Add", aotTestLocal(sum), aotTestLocal(i))
	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs:  []*ast.Node{inc, add},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(1)),
			aotTestBinding(sum, aotTestInt(math.MaxInt64)),
		},
		Body: aotTestIf(
			aotTestNumbersCall("Lt", aotTestLocal(i), aotTestInt(2)),
			recur,
			aotTestLocal(sum),
		),
	}
	method := &ast.FnMethodNode{Body: loop}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 loop was not specialized")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[inc.Sub.(*ast.HostCallNode)] {
		t.Fatal("safe induction increment retained an overflow check")
	}
	if analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] &&
		!analysis.guardInt32Loops[loop.Sub.(*ast.LetNode)] {
		t.Fatal("possibly overflowing accumulator addition lost its check without a range guard")
	}
}

func TestInt64AOTSpeculatesOnlyBehindParameterGuard(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-guarded-proof"))
	vr := ns.Intern(lang.NewSymbol("double"))
	n := lang.NewSymbol("n")
	add := aotTestNumbersCall("Add", aotTestLocal(n), aotTestLocal(n))
	method := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       add,
	}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 function was not specialized")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] {
		t.Fatal("signed-32 addition retained an overflow check")
	}
	if !analysis.guardInt32Params {
		t.Fatal("speculative addition omitted its parameter range guard")
	}
}

func TestInt64AOTFallbackGuardEmission(t *testing.T) {
	var generated bytes.Buffer
	generator := NewGenerator(&generated)
	generator.writeInt32AOTFallbackGuards([]string{"left", "right"})

	source := generated.String()
	for _, name := range []string{"left", "right"} {
		guard := "if " + name + " < -2147483647 || " +
			name + " > 2147483647 {"
		if !strings.Contains(source, guard) {
			t.Fatalf("generated source omitted %q:\n%s", guard, source)
		}
	}
	if got := strings.Count(source, "return 0, false"); got != 2 {
		t.Fatalf("generated source has %d fallbacks, want 2:\n%s", got, source)
	}
}

func TestSnapshotAOTReferencesUsesCompactExclusions(t *testing.T) {
	source := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.snapshot-source"))
	for _, name := range []string{"first", "second", "excluded"} {
		source.Intern(lang.NewSymbol(name))
	}
	refs := []aotReferredVar{
		{symName: "first", srcNS: source.Name().String(), srcSym: "first"},
		{symName: "second", srcNS: source.Name().String(), srcSym: "second"},
	}

	snapshot, exclusions := snapshotAOTReferences(source.Name().String(), refs)
	if !snapshot {
		t.Fatal("dense reference set did not use a shared snapshot")
	}
	if len(exclusions) != 1 || exclusions[0] != "excluded" {
		t.Fatalf("snapshot exclusions = %v, want [excluded]", exclusions)
	}
}

func TestAnalyzeInt64AOTAcrossVarCall(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-cross-call"))
	calleeVar := ns.Intern(lang.NewSymbol("callee"))
	callerVar := ns.Intern(lang.NewSymbol("caller"))
	n := lang.NewSymbol("n")

	calleeTarget := &aotSpecializationTarget{vr: calleeVar, arity: 1}
	calleeMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body: aotTestNumbersCall(
			"Add",
			aotTestLocal(n),
			aotTestInt(1),
		),
	}
	targets := map[*lang.Var]*aotSpecializationTarget{
		calleeVar: calleeTarget,
		callerVar: {vr: callerVar, arity: 1},
	}

	callerMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       aotTestInvoke(calleeVar, aotTestLocal(n)),
	}
	if analysis := analyzeInt64AOTFunction(
		targets[callerVar],
		callerMethod,
		targets,
	); analysis != nil {
		t.Fatal("caller specialized before its callee had a primitive path")
	}

	calleeTarget.int64Analysis = analyzeInt64AOTFunction(
		calleeTarget,
		calleeMethod,
		targets,
		true,
	)
	if calleeTarget.int64Analysis == nil {
		t.Fatal("callee did not receive a primitive path")
	}
	if analysis := analyzeInt64AOTFunction(
		targets[callerVar],
		callerMethod,
		targets,
	); analysis == nil {
		t.Fatal("caller did not specialize after its callee")
	}
}

func TestAnalyzeFloat64AOTMixedLoopAndCrossCall(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.float64-analysis"))
	polynomialVar := ns.Intern(lang.NewSymbol("polynomial"))
	runVar := ns.Intern(lang.NewSymbol("run"))
	x := lang.NewSymbol("x")
	i := lang.NewSymbol("i")
	total := lang.NewSymbol("total")
	loopID := lang.NewSymbol("float-loop")

	polynomialTarget := &aotSpecializationTarget{vr: polynomialVar, arity: 1}
	runTarget := &aotSpecializationTarget{vr: runVar, arity: 0}
	targets := map[*lang.Var]*aotSpecializationTarget{
		polynomialVar: polynomialTarget,
		runVar:        runTarget,
	}
	polynomialMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(x, nil)},
		FixedArity: 1,
		Body: aotTestNumbersCall(
			"Add",
			aotTestNumbersCall("Multiply", aotTestLocal(x), aotTestLocal(x)),
			aotTestConst(float64(0.5)),
		),
	}
	polynomialTarget.float64Analysis = analyzeFloat64AOTFunction(
		polynomialTarget,
		polynomialMethod,
		targets,
	)
	if polynomialTarget.float64Analysis == nil {
		t.Fatal("float64 callee was not specialized")
	}

	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs: []*ast.Node{
			aotTestNumbersCall("Inc", aotTestLocal(i)),
			aotTestNumbersCall(
				"Add",
				aotTestLocal(total),
				aotTestInvoke(polynomialVar, aotTestConst(float64(1.5))),
			),
		},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(0)),
			aotTestBinding(total, aotTestConst(float64(0))),
		},
		Body: aotTestIf(
			aotTestInvoke(
				lang.NSCore.Intern(lang.NewSymbol("=")),
				aotTestLocal(i),
				aotTestInt(10),
			),
			aotTestLocal(total),
			recur,
		),
	}
	runMethod := &ast.FnMethodNode{Body: loop}
	if analysis := analyzeFloat64AOTFunction(
		runTarget,
		runMethod,
		targets,
	); analysis == nil {
		t.Fatal("mixed int64/float64 loop was not specialized")
	}

	analyzer := newFloat64AOTAnalyzer(
		&float64AOTAnalysis{target: runTarget},
		targets,
	)
	mixedEquality := aotTestInvoke(
		lang.NSCore.Intern(lang.NewSymbol("=")),
		aotTestInt(9007199254740993),
		aotTestConst(float64(9007199254740992)),
	)
	if typ := analyzer.exprType(mixedEquality, nil); typ != invalidAOTPrimitive {
		t.Fatalf("mixed numeric equality received unsafe primitive type %v", typ)
	}
}

func TestAnalyzeReducePipeline(t *testing.T) {
	core := lang.NSCore
	coreVar := func(name string) *lang.Var {
		return core.Intern(lang.NewSymbol(name))
	}
	rangeCall := aotTestInvoke(coreVar("range"), aotTestInt(100))
	filterCall := aotTestInvoke(
		coreVar("filter"),
		aotTestVar(coreVar("odd?")),
		rangeCall,
	)
	mapCall := aotTestInvoke(
		coreVar("map"),
		aotTestVar(coreVar("inc")),
		filterCall,
	)
	reduce := aotTestInvoke(
		coreVar("reduce"),
		aotTestVar(coreVar("+")),
		aotTestInt(0),
		mapCall,
	)

	plan := compiler.AnalyzePipeline(reduce)
	if plan == nil || plan.Lowering != compiler.IRPipelineReduceInt64 {
		t.Fatal("safe integer range pipeline was not fused")
	}
	want := []ReducePipelineTransformKind{
		ReducePipelineFilterOdd,
		ReducePipelineMapInc,
	}
	if len(plan.Stages) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(plan.Stages), len(want))
	}
	for i, stage := range plan.Stages {
		if stage.Primitive != want[i] {
			t.Fatalf("transform %d = %v, want %v", i, stage.Primitive, want[i])
		}
	}

	rangeCall.Sub.(*ast.InvokeNode).Args[0] = aotTestLocal(lang.NewSymbol("n"))
	if plan := compiler.AnalyzePipeline(reduce); plan != nil &&
		plan.Lowering == compiler.IRPipelineReduceInt64 {
		t.Fatal("pipeline with an unproven range bound was fused")
	}
}

func TestGenerateInlineIndexedCollectionCallbacks(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.inline-indexed-collections"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn multiply-values [values factor]
		  (mapv (fn [value] (* value factor)) values))
		(defn sum-until-three [values]
		  (reduce
		    (fn [total value]
		      (if (= value 3)
		        (reduced total)
		        (+ total value)))
		    0
		    values))`)

	var output bytes.Buffer
	if err := NewGenerator(&output).Generate(ns); err != nil {
		t.Fatalf("generate indexed collection callbacks: %v", err)
	}
	generated := output.String()
	for _, expected := range []string{
		".(lang.Indexed)",
		".Nth(",
		"lang.IsReduced(",
		".AsTransient().(*lang.TransientVector)",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("inline collection loop omitted %q:\n%s",
				expected, generated)
		}
	}

	sum := ns.FindInternedVar(lang.NewSymbol("sum-until-three"))
	if got := sum.Invoke(lang.NewVector(
		int64(1),
		int64(2),
		int64(3),
		int64(4),
	)); got != int64(3) {
		t.Fatalf("vector reduced result = %v, want 3", got)
	}
	if got := sum.Invoke(lang.NewList(
		int64(1),
		int64(2),
		int64(3),
		int64(4),
	)); got != int64(3) {
		t.Fatalf("fallback reduced result = %v, want 3", got)
	}
}

func TestGenerateOwnedNestedVectorUpdateRegion(t *testing.T) {
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("codegen.owned-nested-vector"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	ReadEval(`
		(defn update-cell [values pair delta]
		  (let [i (nth pair 0)
		        j (nth pair 1)
		        row (nth values i)]
		    (assoc-in values [i j] (+ (nth row j) delta))))
		(defn update-all [values pairs delta]
		  (let [updated
		        (reduce
		          (fn [state pair] (update-cell state pair delta))
		          values
		          pairs)]
		    (mapv
		      (fn [row]
		        (assoc row 0 (+ (nth row 0) delta)))
		      updated)))`)

	var output bytes.Buffer
	generator := NewGenerator(&output)
	if err := generator.Generate(ns); err != nil {
		t.Fatalf("generate owned nested vector region: %v", err)
	}
	for _, name := range []string{"update-cell", "update-all"} {
		vr := ns.FindInternedVar(lang.NewSymbol(name))
		target := generator.aotCallTargets[vr]
		if target == nil || target.ownedVectorAnalysis == nil {
			var dump string
			if target != nil {
				dump = ast.Format(target.fn.ASTNode())
			}
			t.Fatalf(
				"%s did not receive owned-vector analysis (target=%v vector=%v):\n%s",
				name,
				target != nil,
				target != nil && target.vectorAnalysis != nil,
				dump,
			)
		}
	}
	generated := output.String()
	for _, expected := range []string{
		"runtime.NewOwnedVector(",
		"aotOwnedVectorFn",
		".NestedSnapshot(",
		".AssocIn2Copy(",
		".AssocCopy(",
		".Assoc(",
		".(lang.Indexed)",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("owned vector region omitted %q:\n%s",
				expected, generated)
		}
	}

	update := ns.FindInternedVar(lang.NewSymbol("update-all"))
	original := lang.NewVector(
		lang.NewVector(int64(1), int64(2)),
		lang.NewVector(int64(3), int64(4)),
	)
	got := update.Invoke(
		original,
		lang.NewVector(
			lang.NewVector(int64(0), int64(1)),
			lang.NewVector(int64(1), int64(0)),
		),
		int64(10),
	)
	want := lang.NewVector(
		lang.NewVector(int64(11), int64(12)),
		lang.NewVector(int64(23), int64(4)),
	)
	if !lang.Equals(got, want) {
		t.Fatalf("updated value = %v, want %v", got, want)
	}
	if !lang.Equals(
		original,
		lang.NewVector(
			lang.NewVector(int64(1), int64(2)),
			lang.NewVector(int64(3), int64(4)),
		),
	) {
		t.Fatalf("persistent input was mutated: %v", original)
	}

	var fallback bytes.Buffer
	if err := newGenerator(&fallback, false).Generate(ns); err != nil {
		t.Fatalf("generate without direct linking: %v", err)
	}
	if strings.Contains(fallback.String(), "runtime.NewOwnedVector(") {
		t.Fatalf(
			"disabled direct linking retained owned-vector specialization:\n%s",
			fallback.String(),
		)
	}
}

func aotTestBinding(name *lang.Symbol, init *ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpBinding, nil)
	node.Sub = &ast.BindingNode{Name: name, Init: init}
	return node
}

func aotTestLocal(name *lang.Symbol) *ast.Node {
	node := ast.MakeNode(ast.OpLocal, nil)
	node.Sub = &ast.LocalNode{Name: name}
	return node
}

func aotTestConst(value any) *ast.Node {
	node := ast.MakeNode(ast.OpConst, nil)
	node.Sub = &ast.ConstNode{Value: value}
	return node
}

func aotTestInt(value int64) *ast.Node {
	return aotTestConst(value)
}

func aotTestNumbersCall(name string, args ...*ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpHostCall, nil)
	node.Sub = &ast.HostCallNode{
		Target:         aotTestConst(lang.Numbers),
		Method:         lang.NewSymbol(name),
		Args:           args,
		ResolvedMethod: true,
	}
	return node
}

func aotTestInvoke(vr *lang.Var, args ...*ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpInvoke, nil)
	node.Sub = &ast.InvokeNode{Fn: aotTestVar(vr), Args: args}
	return node
}

func aotTestVar(vr *lang.Var) *ast.Node {
	node := ast.MakeNode(ast.OpVar, nil)
	node.Sub = &ast.VarNode{Var: vr}
	return node
}

func aotTestIf(test, then, otherwise *ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpIf, nil)
	node.Sub = &ast.IfNode{Test: test, Then: then, Else: otherwise}
	return node
}
