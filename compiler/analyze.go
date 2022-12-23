package compiler

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/glojurelang/glojure/ast"
	"github.com/glojurelang/glojure/value"
)

// interface {
// 	Macroexpand1(form interface{}) (interface{}, error)
// 	CurrentNamespace() *value.Symbol
// 	Locals() value.IPersistentMap
// 	Context() value.Keyword
// }

var (
	ctxExpr      = kw("ctx/expr")
	ctxReturn    = kw("ctx/return")
	ctxStatement = kw("ctx/statement")
)

type (
	Env value.IPersistentMap

	Analyzer struct {
		Macroexpand1 func(form interface{}) (interface{}, error)
		CreateVar    func(sym *value.Symbol) *value.Var
		IsVar        func(v interface{}) bool
	}
)

// Analyze performs semantic analysis on the given s-expression,
// returning an AST.
func (a *Analyzer) Analyze(form interface{}, env Env) (ast.Node, error) {
	return a.analyzeForm(form, env)
}

func (a *Analyzer) analyzeForm(form interface{}, env Env) (n ast.Node, err error) {
	// defer func() {
	// 	fmt.Printf("analyzed: %v from %v\n", n, form)
	// }()

	switch v := form.(type) {
	case *value.Symbol:
		return a.analyzeSymbol(v, env)
	case value.IPersistentVector:
		return a.analyzeVector(v, env)
	case value.IPersistentMap:
		return a.analyzeMap(v, env)
	case value.IPersistentSet:
		return a.analyzeSet(v, env)
	case value.ISeq:
		return a.analyzeSeq(v, env)
	default:
		return a.analyzeConst(v, env)
	}
}

// analyzeSymbol performs semantic analysis on the given symbol,
// returning an AST.
func (a *Analyzer) analyzeSymbol(form *value.Symbol, env Env) (ast.Node, error) {
	mform, err := a.Macroexpand1(form)
	if err != nil {
		return nil, err
	}
	if !value.Equal(form, mform) {
		n, err := a.analyzeForm(mform, env)
		if err != nil {
			return nil, err
		}
		return withRawForm(n, form), nil
	}

	var n ast.Node
	if localBinding := value.Get(value.Get(env, kw("locals")), form); localBinding != nil {
		mutable := value.Get(localBinding, kw("mutable"))
		children := value.Get(localBinding, kw("children"))
		n = merge(value.Dissoc(localBinding, kw("init")), value.NewMap(
			kw("op"), kw("local"),
			kw("assignable?"), mutable != nil && mutable != false,
			kw("children"), value.NewVectorFromCollection(remove(children, kw("init"))),
		))
	} else {
		v := resolveSym(form, env)
		vr, ok := v.(*value.Var)
		if ok {
			m := vr.Meta()
			n = value.NewMap(
				kw("op"), kw("var"),
				// kw("assignable?"), dynamicVar(vr, m), // TODO
				kw("var"), vr,
				kw("meta"), m,
			)
		} else {
			maybeClass := form.Namespace()
			if maybeClass != "" {
				n = value.NewMap(
					kw("op"), kw("maybe-host-form"), // TODO: define this for Go interop
					kw("class"), maybeClass,
					kw("field"), value.NewSymbol(form.Name()),
				)
			} else {
				n = value.NewMap(
					kw("op"), kw("maybe-class"),
					kw("class"), mform,
				)
			}
		}
	}

	return merge(n, value.NewMap(
		kw("env"), env,
		kw("form"), mform,
	)), nil
}

// analyzeVector performs semantic analysis on the given vector,
// returning an AST.
func (a *Analyzer) analyzeVector(form value.IPersistentVector, env Env) (ast.Node, error) {
	n := ast.MakeNode(kw("vector"), form)
	var items []ast.Node
	for i := 0; i < form.Count(); i++ {
		// TODO: pass an "items-env" with an expr context
		nn, err := a.analyzeForm(value.MustNth(form, i), env)
		if err != nil {
			return nil, err
		}

		items = append(items, nn)
	}
	n = n.Assoc(kw("items"), items)
	return n.Assoc(kw("children"), value.NewVector(kw("items"))), nil
}

// analyzeMap performs semantic analysis on the given map,
// returning an AST.
func (a *Analyzer) analyzeMap(v value.IPersistentMap, env Env) (ast.Node, error) {
	n := ast.MakeNode(kw("map"), v)
	var keys []ast.Node
	var vals []ast.Node
	for seq := value.Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass a "kv-env" with an expr context

		entry := seq.First().(*value.MapEntry)
		keyNode, err := a.analyzeForm(entry.Key, env)
		if err != nil {
			return nil, err
		}
		valNode, err := a.analyzeForm(entry.Val(), env)
		if err != nil {
			return nil, err
		}
		keys = append(keys, keyNode)
		vals = append(vals, valNode)
	}
	n = n.Assoc(kw("keys"), keys).Assoc(kw("vals"), vals)
	return n.Assoc(kw("children"), value.NewVector(kw("keys"), kw("vals"))), nil
}

// analyzeSet performs semantic analysis on the given set,
// returning an AST.
func (a *Analyzer) analyzeSet(v value.IPersistentSet, env Env) (ast.Node, error) {
	n := ast.MakeNode(kw("set"), v)
	items := make([]ast.Node, 0, v.Count())
	for seq := value.Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass an "items-env" with an expr context
		item, err := a.analyzeForm(seq.First(), env)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	n = n.Assoc(kw("items"), items)
	return n.Assoc(kw("children"), value.NewVector(kw("items"))), nil
}

// analyzeSeq performs semantic analysis on the given sequence,
// returning an AST.
func (a *Analyzer) analyzeSeq(form value.ISeq, env Env) (ast.Node, error) {
	op := form.First()
	if op == nil {
		return nil, exInfo("can't call nil", nil) // TODO: include form and source info
	}
	mform, err := a.Macroexpand1(form)
	if err != nil {
		return nil, err
	}

	if value.Equal(form, mform) {
		return a.parse(form, env)
	}
	n, err := a.analyzeForm(mform, env)
	if err != nil {
		return nil, err
	}

	// TODO: add resolved-op to meta
	return withRawForm(n, form), nil
}

// analyzeConst performs semantic analysis on the given constant
// expression,
func (a *Analyzer) analyzeConst(v interface{}, env Env) (ast.Node, error) {
	n := ast.MakeNode(kw("const"), v)
	n = n.Assoc(kw("type"), classifyType(v)).
		Assoc(kw("val"), v).
		Assoc(kw("literal"), fmt.Sprintf("%v", v))

	if im, ok := v.(value.IMeta); ok {
		meta := im.Meta()
		if meta != nil {
			mn, err := a.analyzeConst(meta, env)
			if err != nil {
				return nil, err
			}
			n = n.Assoc(kw("meta"), mn).
				Assoc(kw("children"), value.NewVector(kw("meta")))
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

////////////////////////////////////////////////////////////////////////////////
// Parse

func (a *Analyzer) parse(form interface{}, env Env) (ast.Node, error) {
	op := value.First(form)
	opSym, ok := op.(*value.Symbol)
	if !ok {
		return a.parseInvoke(form, env)
	}
	switch opSym.Name() {
	case "do":
		return a.parseDo(form, env)
	case "if":
		return a.parseIf(form, env)
	case "new":
		return a.parseNew(form, env)
	case "quote":
		return a.parseQuote(form, env)
	case "set!":
		return a.parseSetBang(form, env)
	case "try":
		return a.parseTry(form, env)
	case "throw":
		return a.parseThrow(form, env)
	case "def":
		return a.parseDef(form, env)
	case ".":
		return a.parseDot(form, env)
	case "let*":
		return a.parseLetStar(form, env)
	case "letfn*":
		return a.parseLetfnStar(form, env)
	case "loop*":
		return a.parseLoopStar(form, env)
	case "recur":
		return a.parseRecur(form, env)
	case "fn*":
		return a.parseFnStar(form, env)
	case "var":
		return a.parseVar(form, env)
	}

	return a.parseInvoke(form, env)
}

func (a *Analyzer) parseInvoke(form interface{}, env Env) (ast.Node, error) {
	panic("parseInvoke unimplemented!")
}

func (a *Analyzer) parseDo(form interface{}, env Env) (ast.Node, error) {
	panic("parseDo unimplemented!")
}

func (a *Analyzer) parseIf(form interface{}, env Env) (ast.Node, error) {
	panic("parseIf unimplemented!")
}

func (a *Analyzer) parseNew(form interface{}, env Env) (ast.Node, error) {
	panic("parseNew unimplemented!")
}

func (a *Analyzer) parseQuote(form interface{}, env Env) (ast.Node, error) {
	panic("parseQuote unimplemented!")
}

func (a *Analyzer) parseSetBang(form interface{}, env Env) (ast.Node, error) {
	panic("parseSetBang unimplemented!")
}

func (a *Analyzer) parseTry(form interface{}, env Env) (ast.Node, error) {
	panic("parseTry unimplemented!")
}

func (a *Analyzer) parseThrow(form interface{}, env Env) (ast.Node, error) {
	panic("parseThrow unimplemented!")
}

func (a *Analyzer) parseDef(form interface{}, env Env) (ast.Node, error) {
	symForm := value.First(value.Rest(form))
	expr := value.Rest(value.Rest(form))

	sym, ok := symForm.(*value.Symbol)
	if !ok {
		return nil, exInfo(fmt.Sprintf("first argument to def must be a symbol, got %T", symForm), nil)
	}

	if sym.Namespace() != "" && sym.Namespace() != value.Get(env, kw("ns")).(*value.Symbol).Name() {
		return nil, exInfo("can't def namespace-qualified symbol", nil)
	}

	var args value.Associative
	var doc interface{}
	switch value.Count(expr) {
	case 0:
		// no-op
	case 1:
		args = value.Assoc(args, kw("init"), value.First(expr))
	case 2:
		doc = value.First(expr)
		init := value.First(value.Rest(expr))
		args = value.Assoc(args, kw("init"), init).
			Assoc(kw("doc"), doc)
	default:
		return nil, exInfo("invalid def", nil)
	}
	if doc == nil {
		doc = value.Get(sym.Meta(), kw("doc"))
	}

	//var vr *value.Var

	var meta *value.Map
	var children *value.Map

	n := ast.MakeNode(kw("def"), form)
	return merge(n, value.NewMap(
		kw("env"), env,
		kw("name"), sym,
		// kw("var"), vr, // TODO
	), meta, args, children), nil
}

func (a *Analyzer) parseDot(form interface{}, env Env) (ast.Node, error) {
	panic("parseDot unimplemented!")
}

func (a *Analyzer) parseLetStar(form interface{}, env Env) (ast.Node, error) {
	panic("parseLetStar unimplemented!")
}

func (a *Analyzer) parseLetfnStar(form interface{}, env Env) (ast.Node, error) {
	panic("parseLetfnStar unimplemented!")
}

func (a *Analyzer) parseLoopStar(form interface{}, env Env) (ast.Node, error) {
	panic("parseLoopStar unimplemented!")
}

func (a *Analyzer) parseRecur(form interface{}, env Env) (ast.Node, error) {
	panic("parseRecur unimplemented!")
}

// (defn parse-fn*
//
//	[[op & args :as form] env]
//	(wrapping-meta
//	 (let [[n meths] (if (symbol? (first args))
//	                   [(first args) (next args)]
//	                   [nil (seq args)])
//	       name-expr {:op    :binding
//	                  :env   env
//	                  :form  n
//	                  :local :fn
//	                  :name  n}
//	       e (if n (assoc (assoc-in env [:locals n] (dissoc-env name-expr)) :local name-expr) env)
//	       once? (-> op meta :once boolean)
//	       menv (assoc (dissoc e :in-try) :once once?)
//	       meths (if (vector? (first meths)) (list meths) meths) ;;turn (fn [] ...) into (fn ([]...))
//	       methods-exprs (mapv #(analyze-fn-method % menv) meths)
//	       variadic (seq (filter :variadic? methods-exprs))
//	       variadic? (boolean variadic)
//	       fixed-arities (seq (map :fixed-arity (remove :variadic? methods-exprs)))
//	       max-fixed-arity (when fixed-arities (apply max fixed-arities))]
//	   (when (>= (count variadic) 2)
//	     (throw (ex-info "Can't have more than 1 variadic overload"
//	                     (merge {:variadics (mapv :form variadic)
//	                             :form      form}
//	                            (-source-info form env)))))
//	   (when (not= (seq (distinct fixed-arities)) fixed-arities)
//	     (throw (ex-info "Can't have 2 or more overloads with the same arity"
//	                     (merge {:form form}
//	                            (-source-info form env)))))
//	   (when (and variadic?
//	              (not-every? #(<= (:fixed-arity %)
//	                         (:fixed-arity (first variadic)))
//	                     (remove :variadic? methods-exprs)))
//	     (throw (ex-info "Can't have fixed arity overload with more params than variadic overload"
//	                     (merge {:form form}
//	                            (-source-info form env)))))
//	   (merge {:op              :fn
//	           :env             env
//	           :form            form
//	           :variadic?       variadic?
//	           :max-fixed-arity max-fixed-arity
//	           :methods         methods-exprs
//	           :once            once?}
//	          (when n
//	            {:local name-expr})
//	          {:children (conj (if n [:local] []) :methods)}))))
func (a *Analyzer) parseFnStar(form interface{}, env Env) (ast.Node, error) {
	// op := value.First(form)
	// args := value.Rest(form)
	panic("parseFnStar unimplemented!")
}

func (a *Analyzer) parseVar(form interface{}, env Env) (ast.Node, error) {
	panic("parseVar unimplemented!")
}

// (defn wrapping-meta
//
//	[{:keys [form env] :as expr}]
//	(let [meta (meta form)]
//	  (if (and (obj? form)
//	           (seq meta))
//	    {:op       :with-meta
//	     :env      env
//	     :form     form
//	     :meta     (analyze-form meta (ctx env :ctx/expr))
//	     :expr     (assoc-in expr [:env :context] :ctx/expr)
//	     :children [:meta :expr]}
//	    expr)))
func (a *Analyzer) wrappingMeta(expr ast.Node) (ast.Node, error) {
	form := ast.Form(expr)
	env := value.Get(expr, kw("env")).(Env)
	var meta value.IPersistentMap
	if m, ok := form.(value.IMeta); ok {
		meta = m.Meta()
	}
	if value.Seq(meta) == nil {
		return expr, nil
	}
	metaNode, err := a.analyzeForm(meta, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	var exprNode ast.Node
	// TODO: assoc-in

	n := ast.MakeNode(kw("with-meta"), form)
	return merge(n, value.NewMap(
		kw("env"), env,
		kw("meta"), metaNode,
		kw("expr"), exprNode,
		kw("children"), value.NewVector(kw("meta"), kw("expr")),
	)), nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers

func resolveSym(sym *value.Symbol, env Env) interface{} {
	// var symNS *value.Symbol
	// if sym.Namespace() != "" {
	// 	symNS = value.NewSymbol(sym.Namespace())
	// }
	// var fullNS *value.Namespace

	// TODO
	return nil
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

func merge(as ...interface{}) value.Associative {
	if len(as) == 0 {
		return nil
	}

	var res value.Associative
	i := 0
	for {
		if i >= len(as) {
			return nil
		}
		if as[i] != nil {
			break
		}
		i++
	}
	res = as[i].(value.Associative)
	for _, a := range as[i+1:] {
		for seq := value.Seq(a); seq != nil; seq = seq.Next() {
			entry := seq.First().(value.IMapEntry)
			res = value.Assoc(res, entry.Key(), entry.Val())
		}
	}
	return res
}

func remove(v interface{}, coll interface{}) interface{} {
	if coll == nil {
		return nil
	}
	var items []interface{}
	for seq := value.Seq(coll); seq != nil; seq = seq.Next() {
		if !value.Equal(v, seq.First()) {
			items = append(items, seq.First())
		}
	}
	return value.NewVector(items...)
}

func ctxEnv(env Env, ctx value.Keyword) Env {
	return value.Assoc(env, kw("context"), ctx).(Env)
}
