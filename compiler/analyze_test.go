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

[(def x 42), {:op :def, :form (def x 42), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {}, :keys [], :vals [], :children [:keys :vals]}, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :children [:meta :init]}]

[(def x), {:op :def, :form (def x), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {}, :keys [], :vals [], :children [:keys :vals]}, :children [:meta]}]

[(def x "docstring" 43), {:op :def, :form (def x "docstring" 43), :env {:ns user}, :name x, :var (var user/x), :meta {:op :map, :form {:doc "docstring"}, :keys [{:op :const, :form :doc, :type :keyword, :val :doc, :literal? true}], :vals [{:op :const, :form "docstring", :type :string, :val "docstring", :literal? true}], :children [:keys :vals]}, :init {:op :const, :form 43, :type :number, :val 43, :literal? true}, :doc "docstring", :children [:meta :init]}]

['foo, {:op :quote, :form (quote foo), :expr {:op :const, :form foo, :type :symbol, :val foo, :literal? true}, :env {:ns user}, :literal? true, :children [:expr]}]

[(+ 1 2), {:op :invoke, :form (+ 1 2), :fn {:op :maybe-class, :class +, :env {:ns user, :context :ctx/expr}, :form +}, :args [{:op :const, :form 1, :type :number, :val 1, :literal? true} {:op :const, :form 2, :type :number, :val 2, :literal? true}], :children [:fn :args]}]

[#'user/foo, {:op :the-var, :form (var user/foo), :env {:ns user}, :var (var user/foo)}]

[(fn* [] "Hello"), {:op :fn, :form (fn* [] "Hello"), :env {:ns user}, :variadic? false, :max-fixed-arity 0, :methods [{:op :fn-method, :form ([] "Hello"), :loop-id loop_1, :env {:ns user, :once false}, :variadic? false, :params [], :fixed-arity 0, :body {:op :do, :form (do "Hello"), :env nil, :statements [], :ret {:op :const, :form "Hello", :type :string, :val "Hello", :literal? true}, :children [:statements :ret], :body? true}, :children [:params :body]}], :once false, :children [:methods]}]
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

			if !value.Equal(second(form), ast) {
				t.Fatalf("\nexpected: %v\nbut got:  %v", second(form), ast)
			}
		})
	}
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
