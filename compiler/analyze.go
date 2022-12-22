package compiler

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/glojurelang/glojure/ast"
	"github.com/glojurelang/glojure/value"
)

type Env interface {
	Macroexpand1(form interface{}) (interface{}, error)
}

// Analyze performs semantic analysis on the given s-expression,
// returning an AST.
func Analyze(env Env, form interface{}) (ast.Node, error) {
	return analyzeForm(env, form)
}

func analyzeForm(env Env, form interface{}) (n ast.Node, err error) {
	defer func() {
		fmt.Printf("analyzed: %v from %v\n", n, form)
	}()

	switch v := form.(type) {
	case *value.Symbol:
		return analyzeSymbol(env, v)
	case value.IPersistentVector:
		return analyzeVector(env, v)
	case value.IPersistentMap:
		return analyzeMap(env, v)
	case value.IPersistentSet:
		return analyzeSet(env, v)
	case value.ISeq:
		return analyzeSeq(env, v)
	default:
		return analyzeConst(env, v)
	}
}

// analyzeSymbol performs semantic analysis on the given symbol,
// returning an AST.
func analyzeSymbol(env Env, form *value.Symbol) (ast.Node, error) {
	// (defn analyze-symbol
	//   [sym env]
	//   (let [mform (macroexpand-1 sym env)] ;; t.a.j/macroexpand-1 macroexpands Class/Field into (. Class Field)
	//     (if (= mform sym)
	//       (merge (if-let [{:keys [mutable children] :as local-binding} (-> env :locals sym)] ;; locals shadow globals
	//                (merge (dissoc local-binding :init)                                      ;; avoids useless passes later
	//                       {:op          :local
	//                        :assignable? (boolean mutable)
	//                        :children    (vec (remove #{:init} children))})
	//                (if-let [var (let [v (resolve-sym sym env)]
	//                               (and (var? v) v))]
	//                  (let [m (meta var)]
	//                    {:op          :var
	//                     :assignable? (dynamic? var m) ;; we cannot statically determine if a Var is in a thread-local context
	//                     :var         var              ;; so checking whether it's dynamic or not is the most we can do
	//                     :meta        m})
	//                  (if-let [maybe-class (namespace sym)] ;; e.g. js/foo.bar or Long/MAX_VALUE
	//                    (let [maybe-class (symbol maybe-class)]
	//                      {:op    :maybe-host-form
	//                       :class maybe-class
	//                       :field (symbol (name sym))})
	//                    {:op    :maybe-class ;; e.g. java.lang.Integer or Long
	//                     :class mform})))
	//              {:env  env
	//               :form mform})
	//       (-> (analyze-form mform env)
	//         (update-in [:raw-forms] (fnil conj ()) sym)))))

	mform, err := env.Macroexpand1(form)
	if err != nil {
		return nil, err
	}
	if !value.Equal(form, mform) {
		n, err := analyzeForm(env, mform)
		if err != nil {
			return nil, err
		}
		return withRawForm(n, form), nil
	}

	return nil, nil
}

// analyzeVector performs semantic analysis on the given vector,
// returning an AST.
func analyzeVector(env Env, form value.IPersistentVector) (ast.Node, error) {
	n := ast.MakeNode(kw("vector"), form)
	var items []ast.Node
	for i := 0; i < form.Count(); i++ {
		// TODO: pass an "items-env" with an expr context
		nn, err := analyzeForm(env, value.MustNth(form, i))
		if err != nil {
			return nil, err
		}

		items = append(items, nn)
	}
	n = n.Assoc(kw("items"), items)
	return n.Assoc(kw("children"), value.NewVector([]interface{}{kw("items")})), nil
}

// analyzeMap performs semantic analysis on the given map,
// returning an AST.
func analyzeMap(env Env, v value.IPersistentMap) (ast.Node, error) {
	n := ast.MakeNode(kw("map"), v)
	var keys []ast.Node
	var vals []ast.Node
	for seq := value.Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass a "kv-env" with an expr context

		entry := seq.First().(*value.MapEntry)
		keyNode, err := analyzeForm(env, entry.Key)
		if err != nil {
			return nil, err
		}
		valNode, err := analyzeForm(env, entry.Val())
		if err != nil {
			return nil, err
		}
		keys = append(keys, keyNode)
		vals = append(vals, valNode)
	}
	n = n.Assoc(kw("keys"), keys).Assoc(kw("vals"), vals)
	return n.Assoc(kw("children"), value.NewVector([]interface{}{kw("keys"), kw("vals")})), nil
}

// analyzeSet performs semantic analysis on the given set,
// returning an AST.
func analyzeSet(env Env, v value.IPersistentSet) (ast.Node, error) {
	n := ast.MakeNode(kw("set"), v)
	items := make([]ast.Node, 0, v.Count())
	for seq := value.Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass an "items-env" with an expr context
		item, err := analyzeForm(env, seq.First())
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	n = n.Assoc(kw("items"), items)
	return n.Assoc(kw("children"), value.NewVector([]interface{}{kw("items")})), nil
}

// analyzeSeq performs semantic analysis on the given sequence,
// returning an AST.
func analyzeSeq(env Env, form value.ISeq) (ast.Node, error) {
	op := form.First()
	if op == nil {
		return nil, exInfo("can't call nil", nil) // TODO: include form and source info
	}
	mform, err := env.Macroexpand1(op)
	if err != nil {
		return nil, err
	}

	if value.Equal(form, mform) {
		return parse(env, form)
	}
	n, err := analyzeForm(env, mform)
	if err != nil {
		return nil, err
	}
	fmt.Printf("n: %v, form: %v\n", n, mform)

	// TODO: add resolved-op to meta
	return withRawForm(n, form), nil
}

// analyzeConst performs semantic analysis on the given constant
// expression,
func analyzeConst(env Env, v interface{}) (ast.Node, error) {
	n := ast.MakeNode(kw("const"), v)
	n = n.Assoc(kw("type"), classifyType(v)).
		Assoc(kw("val"), v).
		Assoc(kw("literal"), fmt.Sprintf("%v", v))

	if im, ok := v.(value.IMeta); ok {
		meta := im.Meta()
		if meta != nil {
			mn, err := analyzeConst(env, meta)
			if err != nil {
				return nil, err
			}
			n = n.Assoc(kw("meta"), mn).
				Assoc(kw("children"), value.NewVector([]interface{}{kw("meta")}))
		}
	}
	return n, nil
}

func classifyType(v interface{}) value.Keyword {
	switch v.(type) {
	case nil:
		return kw("nil")
	case bool:
		return kw("bool")
	case value.Keyword:
		return kw("keyword")
	case *value.Symbol:
		return kw("symbol")
	case string:
		return kw("string")
	case value.IPersistentVector:
		return kw("vector")
	case value.IPersistentMap:
		return kw("map")
	case value.IPersistentSet:
		return kw("set")
	case value.ISeq:
		return kw("seq")
	case *value.Char:
		return kw("char")
	case *regexp.Regexp:
		return kw("regex")
	case *value.Var:
		return kw("var")
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		*value.BigInt, *value.BigDecimal, *value.Ratio:
		return kw("number")
	default:
		return kw("unknown")

		// TODO: type, record, class
	}
}

func parse(env Env, _ interface{}) (ast.Node, error) {
	return nil, nil
}

func kw(s string) value.Keyword {
	return value.NewKeyword(s)
}

func exInfo(errStr string, _ interface{}) error {
	// TODO
	return errors.New(errStr)
}

func withRawForm(n ast.Node, form interface{}) ast.Node {
	rawFormsKV := n.EntryAt(kw("raw-forms"))
	if rf, ok := rawFormsKV.Val().(value.Conjer); ok {
		return n.Assoc(kw("raw-forms"), value.Conj(rf, form))
	}
	return n
}
