package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/value"
)

const testForms = `
[42, {:op :const, :form 42, :type :number, :val 42, :literal? true}]

[{}, {:op :map, :form {}, :keys [], :vals [], :children [:keys :vals]}]

[[], {:op :vector, :form [], :items [], :children [:items]}]

[#{}, {:op :set, :form #{}, :items [], :children [:items]}]

[true, {:op :const, :form true, :type :bool, :val true, :literal? true}]

[false, {:op :const, :form false, :type :bool, :val false, :literal? true}]

[(), {:op :const, :form (), :type :seq, :val (), :literal? true}]

[(def x 42),
 {:op :def, :form (def x 42), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {}, :keys [{}], :vals [{}], :children [:keys :vals]}, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :children [:meta :init]}]

[(def x), {:op :def, :form (def x), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {}, :keys [], :vals [], :children [:keys :vals]}, :children [:meta]}]

[(def x "docstring" 43), {:op :def, :form (def x "docstring" 43), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {:doc "docstring"}, :keys [{:op :const, :form :doc, :type :keyword, :val :doc, :literal? true}], :vals [{:op :const, :form "docstring", :type :string, :val "docstring", :literal? true}], :children [:keys :vals]}, :init {:op :const, :form 43, :type :number, :val 43, :literal? true}, :doc "docstring", :children [:meta :init]}]

['foo, {:op :quote, :form (quote foo), :expr {:op :const, :form foo, :type :symbol, :val foo, :literal? true}, :env {:ns user}, :literal? true, :children [:expr]}]

[(+ 1 2), {:op :invoke, :form (+ 1 2), :fn {:op :maybe-class, :class +, :env {:ns user, :context :ctx/expr}, :form +}, :args [{:op :const, :form 1, :type :number, :val 1, :literal? true} {:op :const, :form 2, :type :number, :val 2, :literal? true}], :children [:fn :args]}]

[#'user/foo, {:op :the-var, :form (var user/foo), :env {:ns user}, :var (var user/foo)}]

[(fn* [] "Hello"), {:op :with-meta, :expr {:op :fn, :form (fn* [] "Hello"), :env {:ns user}, :variadic? false, :max-fixed-arity 0, :methods [{:op :fn-method, :form ([] "Hello"), :loop-id loop_1, :env {:ns user, :once false}, :variadic? false, :params [], :fixed-arity 0, :body {:op :do, :form (do "Hello"), :env {:ns user, :once false, :locals {}, :context :ctx/return, :loop-id loop_1, :loop-locals 0}, :statements [], :ret {:op :const, :form "Hello", :type :string, :val "Hello", :literal? true}, :children [:statements :ret], :body? true}, :children [:params :body]}], :once false, :children [:methods]}}]

[(fn* [first & rest] 42) {:op :with-meta, :expr {:op :fn, :form (fn* [first & rest] 42), :env {:ns user}, :variadic? true, :max-fixed-arity 1, :methods [{:op :fn-method, :form ([first & rest] 42), :loop-id loop_1, :env {:ns user, :once false}, :variadic? true, :params [{:env {:ns user, :once false}, :form first, :name first, :variadic? false, :op :binding, :arg-id 0, :local :arg} {:env {:ns user, :once false}, :form rest, :name rest, :variadic? true, :op :binding, :arg-id 1, :local :arg}], :fixed-arity 1, :body {:op :do, :form (do 42), :env {:ns user, :once false, :locals {first {:form first, :name first, :variadic? false, :op :binding, :arg-id 0, :local :arg}, rest {:form rest, :name rest, :variadic? true, :op :binding, :arg-id 1, :local :arg}}, :context :ctx/return, :loop-id loop_1, :loop-locals 2}, :statements [], :ret {:op :const, :form 42, :type :number, :val 42, :literal? true}, :children [:statements :ret], :body? true}, :children [:params :body]}], :once false, :children [:methods]}}]

[(. x Meta)
 {:form (. x Meta), :env {:ns user, :context :ctx/expr, :top-level true}, :target {:op :maybe-class, :class x, :env {:ns user, :context :ctx/expr, :top-level true}, :form x}, :op :host-interop, :assignable? true, :m-or-f Meta, :children [:target]}]

[(if true 42 43) {:op :if, :form (if true 42 43), :env {:ns user}, :test {:op :const, :form true, :type :bool, :val true, :literal? true}, :then {:op :const, :form 42, :type :number, :val 42, :literal? true}, :else {:op :const, :form 43, :type :number, :val 43, :literal? true}, :children [:test :then :else]}]

[(if true 42) {:op :if, :form (if true 42), :env {:ns user}, :test {:op :const, :form true, :type :bool, :val true, :literal? true}, :then {:op :const, :form 42, :type :number, :val 42, :literal? true}, :else {:op :const, :form nil, :type :nil, :val nil, :literal? true}, :children [:test :then :else]}]

[(let* [x 42] x) {:op :let, :form (let* [x 42] x), :env {:ns user}, :body {:op :do, :form (do x), :env {:ns user, :context nil, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :let, :children [:init]}}}, :statements [], :ret {:op :local, :form x, :name x, :local :let, :children [], :assignable? false, :env {:ns user, :context nil, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :let, :children [:init]}}}}, :children [:statements :ret], :body? true}, :bindings [{:op :binding, :form x, :env {:ns user, :context :ctx/expr}, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :let, :children [:init]}], :children [:bindings :body]}]

[(loop* [x 42] x) {:op :loop, :form (loop* [x 42] x), :env {:ns user, :loop-id loop_1}, :loop-id loop_1, :body {:op :do, :form (do x), :env {:ns user, :loop-id loop_1, :context :ctx/return, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}, :statements [], :ret {:op :local, :form x, :name x, :local :loop, :children [], :assignable? false, :env {:ns user, :loop-id loop_1, :context :ctx/return, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}}, :children [:statements :ret], :body? true}, :bindings [{:op :binding, :form x, :env {:ns user, :loop-id loop_1, :context :ctx/expr}, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}], :children [:bindings :body]}]

[(loop* [x 42] (recur (- x 1)))
 {:op :loop, :form (loop* [x 42] (recur (- x 1))), :env {:ns user, :context :ctx/expr, :top-level true, :loop-id loop_1}, :loop-id loop_1, :body {:op :do, :form (do (recur (- x 1))), :env {:ns user, :context :ctx/return, :top-level true, :loop-id loop_1, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}, :statements [], :ret {:op :recur, :env {:ns user, :context :ctx/return, :top-level true, :loop-id loop_1, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}, :form (recur (- x 1)), :exprs [{:op :invoke, :form (- x 1), :fn {:op :maybe-class, :class -, :env {:ns user, :context :ctx/expr, :top-level true, :loop-id loop_1, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}, :form -}, :args [{:op :local, :form x, :name x, :local :loop, :children [], :assignable? false, :env {:ns user, :context :ctx/expr, :top-level true, :loop-id loop_1, :locals {x {:op :binding, :form x, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}}, :loop-locals 1}} {:op :const, :form 1, :type :number, :val 1, :literal? true}], :meta {:file "<unknown-file>", :line 44, :column 23, :end-line 44, :end-column 29}, :children [:fn :args]}], :loop-id loop_1, :children [:exprs]}, :children [:statements :ret], :body? true}, :bindings [{:op :binding, :form x, :env {:ns user, :context :ctx/expr, :top-level true, :loop-id loop_1}, :name x, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :local :loop, :children [:init]}], :children [:bindings :body]}]

[(new strings.Builder) {:op :new, :env {:ns user}, :form (new strings.Builder), :class {:op :maybe-class, :class strings.Builder, :env {:ns user, :locals {}}, :form strings.Builder}, :args [], :children [:class :args]}]

[(throw "not good") {:op :throw, :env {:ns user}, :form (throw "not good"), :exception {:op :const, :form "not good", :type :string, :val "not good", :literal? true}, :children [:exception]}]

[(set! (. Foo -X) 42) {:op :set!, :form (set! (. Foo -X) 42), :env {:ns user}, :target {:form (. Foo -X), :env {:ns user, :context :ctx/expr}, :target {:op :maybe-class, :class Foo, :env {:ns user, :context :ctx/expr}, :form Foo}, :op :host-field, :assignable? true, :field X, :children [:target]}, :val {:op :const, :form 42, :type :number, :val 42, :literal? true}, :children [:target :val]}]

[(try 1) {:op :try}]


[(letfn* [foo (fn* foo ([] 1))])
{:op :letfn, :env {:ns user, :context :ctx/expr, :top-level true}, :form (letfn* [foo (fn* foo ([] 1))]), :bindings ({:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn, :init {:op :with-meta, :form (fn* foo ([] 1)), :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :meta {:op :map, :form {:file "<unknown-file>", :line 56, :column 15, :end-line 56, :end-column 30}, :keys [{:op :const, :form :file, :type :keyword, :val :file, :literal? true} {:op :const, :form :line, :type :keyword, :val :line, :literal? true} {:op :const, :form :column, :type :keyword, :val :column, :literal? true} {:op :const, :form :end-line, :type :keyword, :val :end-line, :literal? true} {:op :const, :form :end-column, :type :keyword, :val :end-column, :literal? true}], :vals [{:op :const, :form "<unknown-file>", :type :string, :val "<unknown-file>", :literal? true} {:op :const, :form 56, :type :number, :val 56, :literal? true} {:op :const, :form 15, :type :number, :val 15, :literal? true} {:op :const, :form 56, :type :number, :val 56, :literal? true} {:op :const, :form 30, :type :number, :val 30, :literal? true}], :children [:keys :vals]}, :expr {:op :fn, :form (fn* foo ([] 1)), :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :variadic? false, :max-fixed-arity 0, :methods [{:op :fn-method, :form ([] 1), :loop-id loop_1, :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :form foo, :local :fn, :name foo}}, :once false}, :variadic? false, :params [], :fixed-arity 0, :body {:op :do, :form (do 1), :env {:ns user, :context :ctx/return, :top-level true, :locals {}, :once false, :loop-id loop_1, :loop-locals 0}, :statements [], :ret {:op :const, :form 1, :type :number, :val 1, :literal? true}, :children [:statements :ret], :body? true}, :children [:params :body]}], :once false, :local {:op :binding, :form foo, :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :local :fn, :name foo}, :children [:local :methods]}, :children [:meta :expr]}, :children [:init]}), :body {:op :do, :form (do), :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :name foo, :form foo, :local :letfn, :init {:op :with-meta, :form (fn* foo ([] 1)), :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :meta {:op :map, :form {:file "<unknown-file>", :line 56, :column 15, :end-line 56, :end-column 30}, :keys [{:op :const, :form :file, :type :keyword, :val :file, :literal? true} {:op :const, :form :line, :type :keyword, :val :line, :literal? true} {:op :const, :form :column, :type :keyword, :val :column, :literal? true} {:op :const, :form :end-line, :type :keyword, :val :end-line, :literal? true} {:op :const, :form :end-column, :type :keyword, :val :end-column, :literal? true}], :vals [{:op :const, :form "<unknown-file>", :type :string, :val "<unknown-file>", :literal? true} {:op :const, :form 56, :type :number, :val 56, :literal? true} {:op :const, :form 15, :type :number, :val 15, :literal? true} {:op :const, :form 56, :type :number, :val 56, :literal? true} {:op :const, :form 30, :type :number, :val 30, :literal? true}], :children [:keys :vals]}, :expr {:op :fn, :form (fn* foo ([] 1)), :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :variadic? false, :max-fixed-arity 0, :methods [{:op :fn-method, :form ([] 1), :loop-id loop_1, :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :form foo, :local :fn, :name foo}}, :once false}, :variadic? false, :params [], :fixed-arity 0, :body {:op :do, :form (do 1), :env {:ns user, :context :ctx/return, :top-level true, :locals {}, :once false, :loop-id loop_1, :loop-locals 0}, :statements [], :ret {:op :const, :form 1, :type :number, :val 1, :literal? true}, :children [:statements :ret], :body? true}, :children [:params :body]}], :once false, :local {:op :binding, :form foo, :env {:ns user, :context :ctx/expr, :top-level true, :locals {foo {:op :binding, :env {:ns user, :context :ctx/expr, :top-level true}, :name foo, :form foo, :local :letfn}}}, :local :fn, :name foo}, :children [:local :methods]}, :children [:meta :expr]}, :children [:init]}}}, :statements [], :ret {:op :const, :form nil, :type :nil, :val nil, :literal? true}, :children [:statements :ret], :body? true}, :children [:bindings :body]}]
`

var (
	globalEnv = value.NewMap(
		kw("namespaces"), value.NewMap(
			value.NewSymbol("user"), value.NewMap(
				kw("ns"), value.NewSymbol("user"),
				kw("mappings"), value.NewMap(
					value.NewSymbol("foo"), value.NewList(value.NewSymbol("var"), value.NewSymbol("user/foo")),
				),
			),
		),
	)
)

func TestAnalyze(t *testing.T) {
	r := reader.New(strings.NewReader(testForms))
	forms, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	const nsName = "user"
	env := value.NewMap(kw("ns"), value.NewSymbol(nsName))

	for _, form := range forms {
		symCounter := 0
		t.Run(value.ToString(value.First(form)), func(t *testing.T) {
			a := &Analyzer{
				Macroexpand1: func(form interface{}) (interface{}, error) {
					return form, nil
				},
				CreateVar: func(sym *value.Symbol, env Env) (interface{}, error) {
					return value.NewList(
						value.NewSymbol("var"),
						value.NewSymbol(nsName+"/"+sym.Name())), nil
				},
				Gensym: func(prefix string) *value.Symbol {
					symCounter++
					return value.NewSymbol(fmt.Sprintf("%s%d", prefix, symCounter))
				},
				GlobalEnv: value.NewAtom(globalEnv),
			}

			ast, err := a.Analyze(value.First(form), env)
			if err != nil {
				t.Fatal(err)
			}

			if d := diff(second(form).(value.IPersistentMap), ast.(value.IPersistentMap)); d != nil {
				t.Errorf("missing keys: %s", d)
				t.Fatalf("\nexpected: %v\nbut got: %v", second(form), ast)
			}
		})
	}
}

func diff(expect, actual interface{}) interface{} {
	switch e := expect.(type) {
	case value.IPersistentMap:
		a, ok := actual.(value.IPersistentMap)
		if !ok {
			return expect
		}
		return diffMap(e, a)
	case value.IPersistentVector:
		a, ok := actual.(value.IPersistentVector)
		if !ok {
			return expect
		}
		return diffVector(e, a)
	default:
		if !value.Equal(expect, actual) {
			return expect
		}
	}
	return nil
}

func diffVector(expect, actual value.IPersistentVector) interface{} {
	// make sure any elements in expect are in actual
	// it's ok if actual has more elements
	for i := 0; i < expect.Count(); i++ {
		found := false
		for j := 0; j < actual.Count(); j++ {
			if diff(value.MustNth(expect, i), value.MustNth(actual, j)) == nil {
				found = true
				break
			}
		}
		if !found {
			return expect
		}
	}
	return nil
}

func diffMap(expect, actual value.IPersistentMap) value.Associative {
	// diffMap recursively checks that any keys in expect are present in actual
	// it's ok for actual to have more keys than expect

	var res value.Associative
	for entrySeq := value.Seq(expect); entrySeq != nil; entrySeq = value.Next(entrySeq) {
		entry := value.First(entrySeq).(value.IMapEntry)
		if !actual.ContainsKey(entry.Key()) {
			res = value.Assoc(res, entry.Key(), entry.Val())
			continue
		}
		actualVal := actual.EntryAt(entry.Key()).Val()
		if av, ok := actualVal.(value.IPersistentMap); ok {
			if ev, ok := entry.Val().(value.IPersistentMap); ok {
				d := diff(ev, av)
				if d != nil {
					res = value.Assoc(res, entry.Key(), d)
				}
			} else {
				res = value.Assoc(res, entry.Key(), entry.Val())
			}
		} else if d := diff(entry.Val(), actualVal); d != nil {
			res = value.Assoc(res, entry.Key(), d)
		}
	}
	return res
}

func FuzzAnalyze(f *testing.F) {
	r := reader.New(strings.NewReader(testForms))
	forms, err := r.ReadAll()
	if err != nil {
		f.Fatal(err)
	}

	for _, form := range forms {
		f.Add(value.ToString(value.First(form)))
	}

	const nsName = "user"
	env := value.NewMap(kw("ns"), value.NewSymbol(nsName))

	f.Fuzz(func(t *testing.T, formStr string) {
		symCounter := 0
		a := &Analyzer{
			Macroexpand1: func(form interface{}) (interface{}, error) {
				return form, nil
			},
			CreateVar: func(sym *value.Symbol, env Env) (interface{}, error) {
				return value.NewList(
					value.NewSymbol("var"),
					value.NewSymbol(nsName+"/"+sym.Name())), nil
			},
			Gensym: func(prefix string) *value.Symbol {
				symCounter++
				return value.NewSymbol(fmt.Sprintf("%s%d", prefix, symCounter))
			},
			GlobalEnv: value.NewAtom(globalEnv),
		}

		r := reader.New(strings.NewReader(formStr))
		form, err := r.ReadOne()
		if err != nil {
			t.Skip()
		}

		// just make sure it doesn't panic
		a.Analyze(form, env)
	})
}
