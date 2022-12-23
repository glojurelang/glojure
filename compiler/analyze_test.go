package compiler

import (
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
[(def x 42), {:op :def, :form (def x 42), :env nil, :name x, :var (var user/x), :meta {:op :map, :form {}, :keys [], :vals [], :children [:keys :vals]}, :init {:op :const, :form 42, :type :number, :val 42, :literal? true}, :children [:meta :init]}]
`

func TestAnalyze(t *testing.T) {
	r := reader.New(strings.NewReader(testForms))
	forms, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	for _, form := range forms {
		const nsName = "user"
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
			}

			ast, err := a.Analyze(value.First(form), nil)
			if err != nil {
				t.Fatal(err)
			}

			if !value.Equal(second(form), ast) {
				t.Fatalf("\nexpected: %v\nbut got:  %v", second(form), ast)
			}
		})
	}
}
