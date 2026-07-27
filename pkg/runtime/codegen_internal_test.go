//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"io/fs"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
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
	).Sub.(*ast.InvokeNode)

	plan := analyzeReducePipeline(reduce)
	if plan == nil {
		t.Fatal("safe integer range pipeline was not fused")
	}
	want := []ReducePipelineTransformKind{
		ReducePipelineFilterOdd,
		ReducePipelineMapInc,
	}
	if len(plan.transforms) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(plan.transforms), len(want))
	}
	for i, transform := range plan.transforms {
		if transform.kind != want[i] {
			t.Fatalf("transform %d = %v, want %v", i, transform.kind, want[i])
		}
	}

	rangeCall.Sub.(*ast.InvokeNode).Args[0] = aotTestLocal(lang.NewSymbol("n"))
	if plan := analyzeReducePipeline(reduce); plan != nil {
		t.Fatal("pipeline with an unproven range bound was fused")
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
