package runtime

import (
	"math"
	"regexp"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

var (
	boxedInt64Sink interface{}
	nativeFnilSink interface{}
)

type testBlockingDeref struct {
	value interface{}
}

func (d testBlockingDeref) Deref() interface{} {
	return d.value
}

func (d testBlockingDeref) DerefWithTimeout(timeoutMS int64, timeoutValue interface{}) interface{} {
	if timeoutMS == 42 {
		return d.value
	}
	return timeoutValue
}

func TestScopeDefineReplacesEquivalentSymbol(t *testing.T) {
	s := newScope()
	first := lang.NewSymbol("value")
	equivalent := lang.NewSymbol("value")

	s.define(first, int64(1))
	s.define(equivalent, int64(2))

	got, ok := s.lookup(first)
	if !ok || got != int64(2) {
		t.Fatalf("lookup = (%v, %v), want (2, true)", got, ok)
	}
}

func TestRTGetCompatibilityMethod(t *testing.T) {
	key := lang.NewKeyword("key")
	m := lang.NewMap(key, int64(42))
	get, ok := lang.FieldOrMethod(RT, "Get")
	if !ok {
		t.Fatal("RT.Get was not resolved")
	}

	if got := lang.Apply2(get, m, key); got != int64(42) {
		t.Fatalf("RT.Get existing key = %v, want 42", got)
	}
	missing := lang.NewKeyword("missing")
	if got := lang.Apply3(get, m, missing, int64(7)); got != int64(7) {
		t.Fatalf("RT.Get missing key = %v, want 7", got)
	}
}

func TestRTCollectionMethodsResolveDirectly(t *testing.T) {
	nth, ok := lang.FieldOrMethod(RT, "Nth")
	if !ok {
		t.Fatal("RT.Nth was not resolved")
	}
	if _, ok := nth.(lang.FnFunc2); !ok {
		t.Fatalf("RT.Nth resolved to %T, want lang.FnFunc2", nth)
	}
	if got := lang.Apply2(nth, lang.NewVector(int64(7)), int64(0)); got != int64(7) {
		t.Fatalf("RT.Nth result = %v, want 7", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		_, _ = lang.FieldOrMethod(RT, "nth")
	}); got != 0 {
		t.Fatalf("cached RT.Nth resolution allocated %v objects per call, want 0", got)
	}
}

func TestNumericLoopRegionSpecializesFloat64AndFallsBack(t *testing.T) {
	x := lang.NewSymbol("x")
	local := &ast.Node{Op: ast.OpLocal, Sub: &ast.LocalNode{Name: x}}
	constant := &ast.Node{Op: ast.OpConst, Sub: &ast.ConstNode{Value: float64(1.5)}}
	call := &ast.HostCallNode{
		Target: &ast.Node{
			Op:  ast.OpConst,
			Sub: &ast.ConstNode{Value: lang.Numbers},
		},
		Method:         lang.NewSymbol("add"),
		Args:           []*ast.Node{local, constant},
		ResolvedMethod: lang.FnFunc2(func(a, b any) any { return lang.Numbers.Add(a, b) }),
	}
	compiler := threadedEvalCompiler{
		typedLoop: true,
		localSlots: map[*lang.Symbol]localSlot{
			x: {
				index:       0,
				kind:        loopLocalSlot,
				numericKind: float64NumericKind,
			},
		},
	}
	evaluator := compiler.compileNumericRegion(call, func(*environment) (interface{}, error) {
		return "fallback", nil
	})
	frame := loopFrame{}
	env := &environment{loopFrame: &frame}

	frame.args[0] = float64(2.5)
	if got, err := evaluator(env); err != nil || got != float64(4) {
		t.Fatalf("float region result = (%v, %v), want (4, nil)", got, err)
	}

	frame.args[0] = int64(2)
	if got, err := evaluator(env); err != nil || got != "fallback" {
		t.Fatalf("mismatched region result = (%v, %v), want (fallback, nil)", got, err)
	}
}

func TestFixedArityTwoFunctionUsesCompiledParameterSlots(t *testing.T) {
	first := lang.NewSymbol("first")
	second := lang.NewSymbol("second")
	params := []*ast.Node{
		{Op: ast.OpBinding, Sub: &ast.BindingNode{Name: first}},
		{Op: ast.OpBinding, Sub: &ast.BindingNode{Name: second}},
	}
	method := &ast.Node{
		Op: ast.OpFnMethod,
		Sub: &ast.FnMethodNode{
			Params:     params,
			FixedArity: 2,
			Body:       &ast.Node{Op: ast.OpLocal, Sub: &ast.LocalNode{Name: second}},
		},
	}
	fn := NewFn(
		&ast.Node{
			Op: ast.OpFn,
			Sub: &ast.FnNode{
				MaxFixedArity: 2,
				Methods:       []*ast.Node{method},
			},
		},
		&environment{scope: newScope()},
	)

	if got := fn.Invoke2("ignored", int64(42)); got != int64(42) {
		t.Fatalf("first invocation = %v, want 42", got)
	}
	if got := fn.Invoke2("ignored again", int64(7)); got != int64(7) {
		t.Fatalf("reused invocation = %v, want 7", got)
	}
}

func TestInterpreterOwnedLoopMapReturnsPersistentValue(t *testing.T) {
	if !testCompilerAvailable {
		t.Skip("source evaluator is unavailable in an AOT runtime build")
	}
	env := NewEnvironment()
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("runtime.owned-loop-map"))
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
		(defn update-map [values]
		  (loop [index 0
		         result values]
		    (if (= index 3)
		      result
		      (recur (inc index)
		             (assoc result index (inc (get result index 0)))))))`,
		WithEnv(env))
	histogram := ns.FindInternedVar(lang.NewSymbol("histogram")).Get()
	result := lang.Apply1(
		histogram,
		lang.NewVector("a", "b", "a", "c", "a"),
	)
	counts, ok := result.(lang.IPersistentMap)
	if !ok {
		t.Fatalf("histogram returned %T, want persistent map", result)
	}
	for key, want := range map[string]int64{"a": 3, "b": 1, "c": 1} {
		if got := counts.ValAt(key); got != want {
			t.Fatalf("histogram[%q] = %v, want %d", key, got, want)
		}
	}

	initial := lang.NewMap(int64(0), int64(10))
	updateMap := ns.FindInternedVar(lang.NewSymbol("update-map")).Get()
	updated := lang.Apply1(updateMap, initial).(lang.IPersistentMap)
	if got := updated.ValAt(int64(0)); got != int64(11) {
		t.Fatalf("adaptive map result[0] = %v, want 11", got)
	}
	if got := updated.ValAt(int64(2)); got != int64(1) {
		t.Fatalf("adaptive map result[2] = %v, want 1", got)
	}
	if got := initial.ValAt(int64(0)); got != int64(10) {
		t.Fatalf("adaptive map loop mutated its input: %v", got)
	}
}

func TestNativeCoreAddApplyToReducibleSequence(t *testing.T) {
	args := lang.NewLongRange(0, 1_000, 1)

	if got := (nativeCoreAdd{}).ApplyTo(args); got != int64(499_500) {
		t.Fatalf("sum = %v, want 499500", got)
	}
}

func TestNativeCoreAddApplyToPreservesUnaryNegativeZero(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	args := lang.NewList(negativeZero)

	got := (nativeCoreAdd{}).ApplyTo(args).(float64)
	if !math.Signbit(got) {
		t.Fatalf("unary sum = %v, want negative zero", got)
	}
}

func TestNativeCoreSubtractApplyToReducibleSequence(t *testing.T) {
	args := lang.NewLongRange(0, 1_000, 1)

	if got := (nativeCoreSubtract{}).ApplyTo(args); got != int64(-499_500) {
		t.Fatalf("difference = %v, want -499500", got)
	}
}

func TestNativeCoreSubtractApplyToPreservesArities(t *testing.T) {
	fn := nativeCoreSubtract{}
	if got := fn.ApplyTo(lang.NewList(int64(5))); got != int64(-5) {
		t.Fatalf("unary difference = %v, want -5", got)
	}
	if got := fn.ApplyTo(lang.NewList(int64(9), int64(4))); got != int64(5) {
		t.Fatalf("binary difference = %v, want 5", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("zero-arity subtraction did not panic")
		}
	}()
	fn.ApplyTo(nil)
}

func TestNativeCoreStrPreservesNilAndStringConversion(t *testing.T) {
	fn := nativeCoreStr{}
	tests := []struct {
		name string
		got  interface{}
		want string
	}{
		{"zero", fn.Invoke0(), ""},
		{"nil", fn.Invoke1(nil), ""},
		{"one", fn.Invoke1(int64(42)), "42"},
		{"two", fn.Invoke2("value=", int64(42)), "value=42"},
		{"three", fn.Invoke3("a", nil, int64(42)), "a42"},
		{"five", fn.Invoke5("a", nil, int64(4), "2", nil), "a42"},
		{"variadic", fn.Invoke("a", nil, int64(42)), "a42"},
		{"apply-to", fn.ApplyTo(lang.NewList("a", nil, int64(42))), "a42"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("str result = %v, want %q", test.got, test.want)
			}
		})
	}

	value := interface{}("already-a-string")
	if got := testing.AllocsPerRun(1_000, func() {
		if fn.Invoke1(value) != value {
			panic("str changed a string")
		}
	}); got != 0 {
		t.Fatalf("one-argument string str allocated %v objects, want 0", got)
	}
}

func TestNativeCoreStrConsumesIndexedMappedSequenceDirectly(t *testing.T) {
	calls := 0
	mapped := nativeMapIndexedSeq(
		lang.FnFunc2(func(index, value interface{}) interface{} {
			calls++
			return lang.ToString(index) + value.(string)
		}),
		lang.NewRepeatN(3, "x"),
		nil,
	)

	if got := (nativeCoreStr{}).ApplyTo(lang.Seq(mapped)); got != "0x1x2x" {
		t.Fatalf("indexed mapped str = %q, want 0x1x2x", got)
	}
	if calls != 3 {
		t.Fatalf("indexed mapping ran %d times, want 3", calls)
	}
}

func TestNativeMapIndexedStringConsumptionCachesRealizedValues(t *testing.T) {
	if !testCompilerAvailable {
		t.Skip("source evaluator is unavailable in an AOT runtime build")
	}
	env := NewEnvironment()
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("runtime.native-map-indexed"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	result := ReadEval(`
		(let [calls (atom 0)
		      values (map-indexed
		               (fn [index value]
		                 (swap! calls inc)
		                 (str index value))
		               (repeat 3 "x"))
		      joined (apply str values)]
		  [joined
		   (second values)
		   @calls
		   (vec (map-indexed (fn [index value] [index value])
		                     ["a" "b"]))
		   (into [] (map-indexed (fn [index value] [index value]))
		         ["a" "b"])])`,
		WithEnv(env),
	).(lang.IPersistentVector)

	if got := result.Nth(0); got != "0x1x2x" {
		t.Fatalf("joined indexed values = %q, want 0x1x2x", got)
	}
	if got := result.Nth(1); got != "1x" {
		t.Fatalf("second cached indexed value = %q, want 1x", got)
	}
	if got := result.Nth(2); got != int64(3) {
		t.Fatalf("indexed callback calls = %v, want 3", got)
	}
	wantIndexed := lang.NewVector(
		lang.NewVector(int64(0), "a"),
		lang.NewVector(int64(1), "b"),
	)
	if got := result.Nth(3); !lang.Equals(got, wantIndexed) {
		t.Fatalf("chunked map-indexed fallback = %v, want %v",
			got, wantIndexed)
	}
	if got := result.Nth(4); !lang.Equals(got, wantIndexed) {
		t.Fatalf("map-indexed transducer fallback = %v, want %v",
			got, wantIndexed)
	}
}

func TestNativeCoreRegexMatchPreservesClojureGroups(t *testing.T) {
	find := nativeCoreRegexMatch{}
	matches := nativeCoreRegexMatch{full: true}

	if got := find.Invoke2(regexp.MustCompile(`b+`), "abbc"); got != "bb" {
		t.Fatalf("re-find scalar = %v, want bb", got)
	}
	if got := matches.Invoke2(regexp.MustCompile(`a(b+)(c)?`), "abb"); !lang.Equals(
		got,
		lang.NewVector("abb", "bb", nil),
	) {
		t.Fatalf("re-matches groups = %v, want [abb bb nil]", got)
	}
	if got := matches.Invoke2(regexp.MustCompile(`b+`), "abbc"); got != nil {
		t.Fatalf("partial re-matches = %v, want nil", got)
	}
}

func TestNativeCoreRegexpSeqThroughCore(t *testing.T) {
	if !testCompilerAvailable {
		t.Skip("source evaluator is unavailable in an AOT runtime build")
	}
	env := NewEnvironment()
	ns := lang.FindOrCreateNamespace(
		lang.NewSymbol("runtime.native-regexp-seq"),
	)
	ns.ReferAllSnapshot(lang.NSCore, nil)
	lang.PushThreadBindings(lang.NewMap(lang.VarCurrentNS, ns))
	defer lang.PopThreadBindings()

	result := ReadEval(`
		(let [matches (re-seq #"[cgt]gggtaaa|tttaccc[acg]"
		                      "xxcgggtaaayytttacccazz")]
		  [(vec matches) (count matches) (counted? matches)])`,
		WithEnv(env),
	).(lang.IPersistentVector)

	if got := result.Nth(0); !lang.Equals(
		got,
		lang.NewVector("cgggtaaa", "tttaccca"),
	) {
		t.Fatalf("core re-seq matches = %v", got)
	}
	if got := lang.MustAsInt(result.Nth(1)); got != 2 {
		t.Fatalf("core re-seq count = %v, want 2", got)
	}
	if got := result.Nth(2); got != false {
		t.Fatalf("core re-seq counted? = %v, want false", got)
	}
}

func TestNativeStringReplaceUsesRegexpSemantics(t *testing.T) {
	fn := nativeStringReplace{}
	for _, test := range []struct {
		pattern     string
		input       string
		replacement string
	}{
		{`a[NSt]|BY`, "aNtaStaStBY", "<2>"},
		{`a+`, "caaab", "_"},
		{`(ab)`, "xxabyy", "$1$1"},
	} {
		expression := regexp.MustCompile(test.pattern)
		want := expression.ReplaceAllString(test.input, test.replacement)
		if got := fn.Invoke3(
			test.input,
			expression,
			test.replacement,
		); got != want {
			t.Fatalf("replace %q = %q, want %q",
				test.pattern, got, want)
		}
	}
}

func TestNativeCoreGetInPreservesMissingAndNilValues(t *testing.T) {
	fn := nativeCoreGetIn{}
	value := lang.NewMap(
		lang.NewKeyword("outer"), lang.NewMap(
			lang.NewKeyword("value"), int64(42),
			lang.NewKeyword("nil"), nil,
		),
	)
	valuePath := lang.NewVector(lang.NewKeyword("outer"), lang.NewKeyword("value"))
	nilPath := lang.NewVector(lang.NewKeyword("outer"), lang.NewKeyword("nil"))
	missingPath := lang.NewVector(lang.NewKeyword("outer"), lang.NewKeyword("missing"))

	if got := fn.Invoke2(value, valuePath); got != int64(42) {
		t.Fatalf("nested value = %v, want 42", got)
	}
	if got := fn.Invoke3(value, nilPath, "missing"); got != nil {
		t.Fatalf("present nil value = %v, want nil", got)
	}
	if got := fn.Invoke3(value, missingPath, "missing"); got != "missing" {
		t.Fatalf("missing value = %v, want missing", got)
	}
	if got := fn.Invoke2(value, nil); got != value {
		t.Fatalf("empty path = %v, want original value", got)
	}
}

func TestNativeCoreAssocHandlesFixedAndVariadicPairs(t *testing.T) {
	fn := nativeCoreAssoc{}
	a, b := lang.NewKeyword("a"), lang.NewKeyword("b")

	if got := fn.Invoke3(nil, a, int64(1)); !lang.Equals(
		got,
		lang.NewMap(a, int64(1)),
	) {
		t.Fatalf("fixed assoc = %v", got)
	}
	if got := fn.Invoke(nil, a, int64(1), b, int64(2)); !lang.Equals(
		got,
		lang.NewMap(a, int64(1), b, int64(2)),
	) {
		t.Fatalf("variadic assoc = %v", got)
	}
	if got := fn.ApplyTo(lang.NewList(nil, a, int64(1), b, int64(2))); !lang.Equals(
		got,
		lang.NewMap(a, int64(1), b, int64(2)),
	) {
		t.Fatalf("sequence assoc = %v", got)
	}
}

func TestNativeStringPrimitives(t *testing.T) {
	if got := (nativeStringIncludes{}).Invoke2("alpha_beta", "_"); got != true {
		t.Fatalf("includes? = %v, want true", got)
	}

	fallbackCalls := 0
	fallback := lang.FnFunc3(func(_, _, _ interface{}) interface{} {
		fallbackCalls++
		return "fallback"
	})
	replace := nativeStringReplace{fallback: fallback}
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"string", replace.Invoke3("a-b-a", "a", "x"), "x-b-x"},
		{"char", replace.Invoke3("a-b-a", lang.Char('a'), lang.Char('x')), "x-b-x"},
		{"regexp", replace.Invoke3("ab12", regexp.MustCompile(`[0-9]+`), "x"), "abx"},
		{"function fallback", replace.Invoke3("ab", regexp.MustCompile(`.`), fallback), "fallback"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s replace = %v, want %v", test.name, test.got, test.want)
		}
	}
	if fallbackCalls != 1 {
		t.Fatalf("replace fallback called %d times, want 1", fallbackCalls)
	}
}

func TestNativeCoreDerefUsesInterfacesAndFallback(t *testing.T) {
	fallbackCalls := 0
	fallback := lang.FnFunc(func(args ...interface{}) interface{} {
		fallbackCalls++
		return "fallback"
	})
	fn := nativeCoreDeref{fallback: fallback}
	value := testBlockingDeref{value: "ready"}

	if got := fn.Invoke1(value); got != "ready" {
		t.Fatalf("one-arity deref = %v, want ready", got)
	}
	if got := fn.Invoke3(value, int64(42), "timeout"); got != "ready" {
		t.Fatalf("timed deref = %v, want ready", got)
	}
	if got := fn.Invoke3(value, int64(7), "timeout"); got != "timeout" {
		t.Fatalf("timed-out deref = %v, want timeout", got)
	}
	if got := fn.Invoke1(struct{}{}); got != "fallback" || fallbackCalls != 1 {
		t.Fatalf("fallback deref = %v with %d calls", got, fallbackCalls)
	}
}

func TestNativeCoreSwapUsesFixedAtomArities(t *testing.T) {
	fallback := lang.FnFunc(func(args ...interface{}) interface{} {
		t.Fatalf("unexpected swap! fallback: %v", args)
		return nil
	})
	fn := nativeCoreSwap{fallback: fallback}
	atom := lang.NewAtom(int64(1))
	add1 := lang.FnFunc1(func(value interface{}) interface{} {
		return lang.Numbers.Add(value, int64(1))
	})
	add := lang.FnFunc2(func(a, b interface{}) interface{} {
		return lang.Numbers.Add(a, b)
	})
	add3 := lang.FnFunc3(func(a, b, c interface{}) interface{} {
		return lang.Numbers.Add(lang.Numbers.Add(a, b), c)
	})

	if got := fn.Invoke2(atom, add1); got != int64(2) {
		t.Fatalf("zero-extra-arg swap = %v, want 2", got)
	}
	if got := fn.Invoke3(atom, add, int64(3)); got != int64(5) {
		t.Fatalf("one-extra-arg swap = %v, want 5", got)
	}
	if got := fn.Invoke4(atom, add3, int64(4), int64(5)); got != int64(14) {
		t.Fatalf("two-extra-arg swap = %v, want 14", got)
	}
	if got := fn.ApplyTo(lang.NewList(atom, add, int64(6))); got != int64(20) {
		t.Fatalf("ApplyTo swap = %v, want 20", got)
	}
}

func TestNativeCoreReduceUsesReductionInterfaces(t *testing.T) {
	fallback := lang.FnFunc(func(args ...interface{}) interface{} {
		t.Fatalf("unexpected reduce fallback: %v", args)
		return nil
	})
	fn := nativeCoreReduce{fallback: fallback}
	add := lang.FnFunc2(func(a, b interface{}) interface{} {
		return lang.Numbers.Add(a, b)
	})
	values := lang.NewVector(int64(1), int64(2), int64(3))

	if got := fn.Invoke2(add, values); got != int64(6) {
		t.Fatalf("two-arity reduce = %v, want 6", got)
	}
	if got := fn.Invoke3(add, int64(4), values); got != int64(10) {
		t.Fatalf("three-arity reduce = %v, want 10", got)
	}
}

func TestNativeCoreReduceUsesNativeArraysAndSlices(t *testing.T) {
	fallback := lang.FnFunc(func(args ...interface{}) interface{} {
		t.Fatalf("unexpected reduce fallback: %v", args)
		return nil
	})
	fn := nativeCoreReduce{fallback: fallback}
	add := lang.FnFunc2(func(a, b interface{}) interface{} {
		return lang.Numbers.Add(a, b)
	})

	if got := fn.Invoke2(add, []int64{1, 2, 3}); got != int64(6) {
		t.Fatalf("slice reduce = %v, want 6", got)
	}
	if got := fn.Invoke3(add, int64(4), [3]int64{1, 2, 3}); got != int64(10) {
		t.Fatalf("array reduce = %v, want 10", got)
	}

	stopAfterTwo := lang.FnFunc2(func(a, b interface{}) interface{} {
		sum := lang.Numbers.Add(a, b)
		if sum == int64(3) {
			return lang.NewReduced(sum)
		}
		return sum
	})
	if got := fn.Invoke3(stopAfterTwo, int64(0), []int64{1, 2, 100}); got != int64(3) {
		t.Fatalf("reduced slice = %v, want 3", got)
	}

	called := false
	empty := lang.FnFunc(func(args ...interface{}) interface{} {
		called = true
		if len(args) != 0 {
			t.Fatalf("empty reduction received args: %v", args)
		}
		return int64(42)
	})
	if got := fn.Invoke2(empty, []int64{}); got != int64(42) || !called {
		t.Fatalf("empty slice reduce = %v, called %v; want 42, true", got, called)
	}
}

func TestNativeCoreRequireFastPathPreservesRuntimeSemantics(t *testing.T) {
	ns := lang.NewNamespace(lang.NewSymbol("runtime.native-require-test"))
	loadedRef := lang.NewRef(lang.NewSet())
	loadedLibs := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("*loaded-libs*"),
		loadedRef,
	).SetDynamic()
	loadingVerbosely := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("*loading-verbosely*"),
		false,
	).SetDynamic()
	loadCalls := 0
	var loadedResource string
	var loadingVerboselyDuringLoad interface{}
	load := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("load"),
		lang.FnFunc1(func(resource interface{}) interface{} {
			loadCalls++
			loadedResource = resource.(string)
			loadingVerboselyDuringLoad = loadingVerbosely.Deref()
			loadingVerbosely.Set(true)
			lang.FindOrCreateNamespace(lang.NewSymbol("native-require.lib-name"))
			return nil
		}),
	)
	contains := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("contains?"),
		lang.FnFunc2(func(coll, value interface{}) interface{} {
			return coll.(*lang.Set).Contains(value)
		}),
	)
	conj := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("conj"),
		lang.FnFunc2(func(coll, value interface{}) interface{} {
			return coll.(*lang.Set).Cons(value)
		}),
	)
	fallbackCalls := 0
	fallback := lang.FnFunc(func(args ...interface{}) interface{} {
		fallbackCalls++
		return "fallback"
	})
	require := nativeCoreRequire{
		fallback:         fallback,
		load:             load,
		loadedLibs:       loadedLibs,
		loadingVerbosely: loadingVerbosely,
		contains:         contains,
		conj:             conj,
	}
	lib := lang.NewSymbol("native-require.lib-name")
	defer lang.RemoveNamespace(lib)

	if got := require.Invoke1(lib); got != nil {
		t.Fatalf("require result = %v, want nil", got)
	}
	if loadedResource != "/native_require/lib_name" {
		t.Fatalf("loaded resource = %q, want /native_require/lib_name", loadedResource)
	}
	if loadCalls != 1 {
		t.Fatalf("load calls = %d, want 1", loadCalls)
	}
	if loadingVerbosely.Deref() != false {
		t.Fatalf("*loading-verbosely* leaked value %v", loadingVerbosely.Deref())
	}
	if loadingVerboselyDuringLoad != nil {
		t.Fatalf(
			"*loading-verbosely* during load = %v, want nil",
			loadingVerboselyDuringLoad,
		)
	}
	if !loadedRef.Deref().(*lang.Set).Contains(lib) {
		t.Fatal("required lib was not recorded")
	}
	if got := require.Invoke1(lib); got != nil || loadCalls != 1 {
		t.Fatalf("second require = %v with %d load calls, want nil and 1", got, loadCalls)
	}
	if got := require.Invoke1(lang.NewVector(lib)); got != "fallback" ||
		fallbackCalls != 1 {
		t.Fatalf("non-symbol require = %v with %d fallback calls", got, fallbackCalls)
	}
}

func TestNativeCoreRequireRemovesNewNamespaceAfterLoadFailure(t *testing.T) {
	ns := lang.NewNamespace(lang.NewSymbol("runtime.native-require-failure-test"))
	lib := lang.NewSymbol("native-require.failure")
	loadedLibs := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("*loaded-libs*"),
		lang.NewRef(lang.NewSet()),
	).SetDynamic()
	loadingVerbosely := lang.NewVarWithRoot(
		ns,
		lang.NewSymbol("*loading-verbosely*"),
		false,
	).SetDynamic()
	require := nativeCoreRequire{
		fallback: lang.FnFunc(func(args ...interface{}) interface{} {
			return nil
		}),
		load: lang.NewVarWithRoot(
			ns,
			lang.NewSymbol("load"),
			lang.FnFunc1(func(interface{}) interface{} {
				lang.FindOrCreateNamespace(lib)
				panic("load failed")
			}),
		),
		loadedLibs:       loadedLibs,
		loadingVerbosely: loadingVerbosely,
		contains: lang.NewVarWithRoot(
			ns,
			lang.NewSymbol("contains?"),
			lang.FnFunc2(func(coll, value interface{}) interface{} {
				return coll.(*lang.Set).Contains(value)
			}),
		),
		conj: lang.NewVarWithRoot(
			ns,
			lang.NewSymbol("conj"),
			lang.FnFunc2(func(coll, value interface{}) interface{} {
				return coll.(*lang.Set).Cons(value)
			}),
		),
	}

	defer func() {
		if recovered := recover(); recovered != "load failed" {
			t.Fatalf("recovered %v, want load failed", recovered)
		}
		if got := lang.FindNamespace(lib); got != nil {
			t.Fatalf("failed require left namespace %v", got)
		}
	}()
	require.Invoke1(lib)
}

func TestNativeCoreApplyPreservesLeadingArguments(t *testing.T) {
	capture := lang.FnFunc(func(args ...any) any {
		return lang.NewVector(args...)
	})
	apply := nativeCoreApply{}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{
			name: "no leading arguments",
			got:  apply.Invoke2(capture, lang.NewVector(int64(1), int64(2))),
			want: lang.NewVector(int64(1), int64(2)),
		},
		{
			name: "one leading argument",
			got:  apply.Invoke3(capture, int64(1), lang.NewVector(int64(2), int64(3))),
			want: lang.NewVector(int64(1), int64(2), int64(3)),
		},
		{
			name: "two leading arguments",
			got:  apply.Invoke4(capture, int64(1), int64(2), lang.NewVector(int64(3), int64(4))),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4)),
		},
		{
			name: "variadic leading arguments",
			got: apply.Invoke(
				capture,
				int64(1),
				int64(2),
				int64(3),
				int64(4),
				lang.NewVector(int64(5), int64(6)),
			),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)),
		},
		{
			name: "apply-to",
			got: apply.ApplyTo(lang.NewList(
				capture,
				int64(1),
				int64(2),
				lang.NewVector(int64(3), int64(4)),
			)),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !lang.Equals(test.got, test.want) {
				t.Fatalf("apply result = %v, want %v", test.got, test.want)
			}
		})
	}
}

func TestNativeCoreUpdateInPreservesCollectionAndVarSemantics(t *testing.T) {
	testNS := lang.FindOrCreateNamespace(lang.NewSymbol("runtime.native-core-test"))
	assocCalls := 0
	applyCalls := 0
	assocVar := lang.NewVarWithRoot(
		testNS,
		lang.NewSymbol("assoc"),
		lang.FnFunc3(func(m, key, value any) any {
			assocCalls++
			return lang.Assoc(m, key, value)
		}),
	)
	applyVar := lang.NewVarWithRoot(
		testNS,
		lang.NewSymbol("apply"),
		lang.FnFunc3(func(fn, value, args any) any {
			applyCalls++
			return lang.ApplySeq(fn, lang.NewCons(value, lang.Seq(args)))
		}),
	)
	updateIn := nativeCoreUpdateIn{assoc: assocVar, apply: applyVar}

	outer := lang.NewKeyword("outer")
	inner := lang.NewKeyword("inner")
	original := lang.NewMap(outer, lang.NewMap(inner, int64(2)))
	add := lang.FnFunc2(func(a, b any) any {
		return lang.Numbers.Add(a, b)
	})

	updated := updateIn.Invoke4(
		original,
		lang.NewVector(outer, inner),
		add,
		int64(3),
	)
	if got := lang.Get(lang.Get(updated, outer), inner); got != int64(5) {
		t.Fatalf("updated value = %v, want 5", got)
	}
	if got := lang.Get(lang.Get(original, outer), inner); got != int64(2) {
		t.Fatalf("original value changed to %v", got)
	}
	if assocCalls != 2 {
		t.Fatalf("assoc Var called %d times, want 2", assocCalls)
	}
	if applyCalls != 1 {
		t.Fatalf("apply Var called %d times, want 1", applyCalls)
	}

	created := updateIn.Invoke3(
		nil,
		lang.NewList(outer, inner),
		lang.FnFunc1(func(value any) any {
			if value != nil {
				t.Fatalf("missing leaf value = %v, want nil", value)
			}
			return int64(7)
		}),
	)
	if got := lang.Get(lang.Get(created, outer), inner); got != int64(7) {
		t.Fatalf("created value = %v, want 7", got)
	}
}

func TestNativeCoreFnilMatchesClojureArities(t *testing.T) {
	capture := lang.FnFunc(func(args ...interface{}) interface{} {
		return lang.NewVector(args...)
	})
	fnil := nativeCoreFnil{}

	one := fnil.Invoke2(capture, "x").(lang.IFn)
	if got, want := one.Invoke(nil), lang.NewVector("x"); !lang.Equals(got, want) {
		t.Fatalf("one-default fnil = %v, want %v", got, want)
	}
	if got, want := one.Invoke(nil, "b", "c", "d", "e"),
		lang.NewVector("x", "b", "c", "d", "e"); !lang.Equals(got, want) {
		t.Fatalf("variadic one-default fnil = %v, want %v", got, want)
	}
	args := []interface{}{nil, "b", "c", "d", "e"}
	one.Invoke(args...)
	if args[0] != nil {
		t.Fatalf("variadic fnil mutated its caller's argument slice: %v", args)
	}

	two := fnil.Invoke3(capture, "x", "y").(lang.IFn)
	if got, want := two.Invoke(nil, nil), lang.NewVector("x", "y"); !lang.Equals(got, want) {
		t.Fatalf("two-default fnil = %v, want %v", got, want)
	}

	three := fnil.Invoke4(capture, "x", "y", "z").(lang.IFn)
	if got, want := three.Invoke(nil, nil, nil),
		lang.NewVector("x", "y", "z"); !lang.Equals(got, want) {
		t.Fatalf("three-default fnil = %v, want %v", got, want)
	}
}

func TestNativeCoreFnilAllocatesOneWrapper(t *testing.T) {
	identity := lang.FnFunc1(func(value interface{}) interface{} { return value })
	allocs := testing.AllocsPerRun(1_000, func() {
		nativeFnilSink = (nativeCoreFnil{}).Invoke2(identity, int64(0))
	})
	if allocs > 1 {
		t.Fatalf("native fnil allocated %v objects, want at most one wrapper", allocs)
	}
}

func TestOwnedLoopMapPreservesPersistentInput(t *testing.T) {
	key := lang.NewKeyword("key")
	other := lang.NewKeyword("other")
	initial := lang.NewMap(key, int64(1))
	loopMap := NewOwnedLoopMap(initial)

	loopMap.Assoc(key, int64(2)).Assoc(other, int64(3))
	result := loopMap.Persistent()

	if got := lang.Get(initial, key); got != int64(1) {
		t.Fatalf("owned loop map mutated its input: %v", got)
	}
	if got := lang.Get(result, other); got != int64(3) {
		t.Fatalf("owned loop map result = %v, want 3", got)
	}
	if got := lang.Get(result, key); got != int64(2) {
		t.Fatalf("owned loop map update = %v, want 2", got)
	}
}

func TestOwnedLoopMapPreservesUnchangedIdentityAndMetadata(t *testing.T) {
	key := lang.NewKeyword("key")
	meta := lang.NewMap(lang.NewKeyword("source"), "test")
	initial := lang.NewMap(key, int64(1)).(lang.IObj).
		WithMeta(meta).(lang.IPersistentMap)

	unchanged := NewOwnedLoopMap(initial)
	unchanged.Assoc(key, int64(1))
	if got := unchanged.Persistent(); got != initial {
		t.Fatalf("no-op owned map update changed identity: %T", got)
	}

	changed := NewOwnedLoopMap(initial)
	changed.Assoc(key, int64(2)).Assoc(lang.NewKeyword("other"), int64(3))
	result := changed.Persistent()
	if got := result.(lang.IMeta).Meta(); got != meta {
		t.Fatalf("owned loop map metadata = %v, want %v", got, meta)
	}
}

func TestOwnedLoopMapKeepsNonMapValuesOnPersistentPath(t *testing.T) {
	vector := lang.NewVector(int64(1), int64(2))
	loopMap := NewOwnedLoopMap(vector)
	loopMap.Assoc(int64(1), int64(3))

	result := loopMap.Persistent()
	if got := result.(*lang.Vector).Nth(1); got != int64(3) {
		t.Fatalf("owned loop vector fallback = %v, want 3", got)
	}
	if got := vector.Nth(1); got != int64(2) {
		t.Fatalf("owned loop map mutated fallback input: %v", got)
	}
}

func TestReduceOwnedMapKeepsNestedUpdatesPrivateUntilReturn(t *testing.T) {
	service := lang.NewKeyword("api")
	requests := lang.NewKeyword("requests")
	initialNested := lang.NewMap(requests, int64(10))
	initial := lang.NewMap(service, initialNested)
	increment := lang.FnFunc1(func(value interface{}) interface{} {
		return lang.Numbers.Inc(value)
	})
	path := lang.NewVector(service, requests)
	reducer := lang.FnFunc2(func(result, _ interface{}) interface{} {
		return UpdateOwnedMap3(result, path, increment)
	})

	result := ReduceOwnedMap(
		nativeCoreReduce{},
		reducer,
		initial,
		lang.NewVector(nil, nil, nil),
	)

	if got := lang.Get(lang.Get(result, service), requests); got != int64(13) {
		t.Fatalf("owned nested result = %v, want 13", got)
	}
	if got := lang.Get(lang.Get(initial, service), requests); got != int64(10) {
		t.Fatalf("owned reduction mutated its persistent input: %v", got)
	}
	if _, ok := result.(lang.IPersistentMap); !ok {
		t.Fatalf("owned reduction leaked %T instead of a persistent map", result)
	}
}

func TestUpdateOwnedMapFixedTwoKeyPathPreservesUpdateSemantics(t *testing.T) {
	service := lang.NewKeyword("api")
	requests := lang.NewKeyword("requests")
	failures := lang.NewKeyword("failures")
	bytes := lang.NewKeyword("bytes")
	initial := lang.NewMap(
		service,
		lang.NewMap(requests, int64(10)),
	)
	increment := lang.FnFunc1(func(value interface{}) interface{} {
		return lang.Numbers.Inc(value)
	})
	add := lang.FnFunc2(func(value, amount interface{}) interface{} {
		if value == nil {
			value = int64(0)
		}
		return lang.Numbers.Add(value, amount)
	})
	reducer := lang.FnFunc2(func(result, _ interface{}) interface{} {
		result = UpdateOwnedMapPath2_3(
			result,
			service,
			requests,
			increment,
		)
		result = UpdateOwnedMapPath2Default3(
			result,
			service,
			failures,
			increment,
			int64(0),
		)
		return UpdateOwnedMapPath2Default4(
			result,
			service,
			bytes,
			add,
			int64(0),
			int64(7),
		)
	})

	result := ReduceOwnedMap(
		nativeCoreReduce{},
		reducer,
		initial,
		lang.NewVector(nil, nil, nil),
	)

	nested := lang.Get(result, service)
	if got := lang.Get(nested, requests); got != int64(13) {
		t.Fatalf("fixed-path requests = %v, want 13", got)
	}
	if got := lang.Get(nested, failures); got != int64(3) {
		t.Fatalf("fixed-path failures = %v, want 3", got)
	}
	if got := lang.Get(nested, bytes); got != int64(21) {
		t.Fatalf("fixed-path bytes = %v, want 21", got)
	}
	if got := lang.Get(lang.Get(initial, service), requests); got != int64(10) {
		t.Fatalf("fixed-path reduction mutated initial map: %v", got)
	}
}

func TestUpdateOwnedMapPassesPersistentMapAtLeaf(t *testing.T) {
	outer := lang.NewKeyword("outer")
	leaf := lang.NewKeyword("leaf")
	innerKey := lang.NewKeyword("inner")
	initialLeaf := lang.NewMap(innerKey, int64(1))
	initial := lang.NewMap(outer, lang.NewMap(leaf, initialLeaf))
	var observed interface{}
	capture := lang.FnFunc1(func(value interface{}) interface{} {
		observed = value
		return value
	})
	reducer := lang.FnFunc2(func(result, _ interface{}) interface{} {
		return UpdateOwnedMap3(
			result,
			lang.NewVector(outer, leaf),
			capture,
		)
	})

	result := ReduceOwnedMap(
		nativeCoreReduce{},
		reducer,
		initial,
		lang.NewVector(nil),
	)
	if _, ok := observed.(lang.IPersistentMap); !ok {
		t.Fatalf("update callback observed transient representation %T", observed)
	}
	if !lang.Equals(lang.Get(lang.Get(result, outer), leaf), initialLeaf) {
		t.Fatalf("leaf map changed: %v", result)
	}
}

func TestReduceOwnedMapPreservesNestedMapMetadata(t *testing.T) {
	outer := lang.NewKeyword("outer")
	leaf := lang.NewKeyword("leaf")
	metaKey := lang.NewKeyword("source")
	meta := lang.NewMap(metaKey, "test")
	nested := lang.NewMap(leaf, int64(1)).(lang.IObj).WithMeta(meta)
	initial := lang.NewMap(outer, nested)
	reducer := lang.FnFunc2(func(result, _ interface{}) interface{} {
		return UpdateOwnedMap3(
			result,
			lang.NewVector(outer, leaf),
			lang.FnFunc1(func(value interface{}) interface{} {
				return lang.Numbers.Inc(value)
			}),
		)
	})

	result := ReduceOwnedMap(
		nativeCoreReduce{},
		reducer,
		initial,
		lang.NewVector(nil),
	)
	got := lang.Get(result, outer).(lang.IMeta).Meta()
	if !lang.Equals(got, meta) {
		t.Fatalf("nested metadata = %v, want %v", got, meta)
	}
}

func TestBoxInt64CachesCommonValues(t *testing.T) {
	for _, value := range []int64{-128, 0, 4096, 4097} {
		if got := boxInt64(value); got != value {
			t.Fatalf("boxInt64(%d) = %v", value, got)
		}
	}
	if got := testing.AllocsPerRun(1_000, func() {
		boxedInt64Sink = boxInt64(42)
	}); got != 0 {
		t.Fatalf("boxInt64 cached value allocated %v objects per call, want 0", got)
	}
}

func TestNativeModMatchesFloorModSemantics(t *testing.T) {
	tests := []struct {
		num  interface{}
		div  interface{}
		want interface{}
	}{
		{num: int64(5), div: int64(3), want: int64(2)},
		{num: int64(-5), div: int64(3), want: int64(1)},
		{num: int64(5), div: int64(-3), want: int64(-1)},
		{num: int64(-5), div: int64(-3), want: int64(-2)},
		{num: float64(-5.5), div: float64(3), want: float64(0.5)},
	}
	for _, test := range tests {
		if got := nativeMod(test.num, test.div); !lang.Equals(got, test.want) {
			t.Errorf("mod(%v, %v) = %v, want %v", test.num, test.div, got, test.want)
		}
	}
}

func TestNativeMapvPreservesOrder(t *testing.T) {
	got := nativeMapv(
		lang.FnFunc1(func(value any) any {
			return lang.Numbers.Multiply(value, int64(2))
		}),
		lang.NewLongRange(0, 5, 1),
	)
	want := lang.NewVector(int64(0), int64(2), int64(4), int64(6), int64(8))
	if !lang.Equals(got, want) {
		t.Fatalf("mapv result = %v, want %v", got, want)
	}
}
