package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"

	// Make it easier to refer to global vars.
	. "github.com/glojurelang/glojure/pkg/lang"
)

var (
	ctxExpr      = KWCtxExpr
	ctxReturn    = KWCtxReturn
	ctxStatement = KWCtxStatement

	symCatch    = NewSymbol("catch")
	symFinally  = NewSymbol("finally")
	symLoopStar = NewSymbol("loop*")
)

type (
	Env IPersistentMap

	Analyzer struct {
		Macroexpand1 func(form interface{}) (interface{}, error)
		CreateVar    func(sym *Symbol, env Env) (interface{}, error)
		IsVar        func(v interface{}) bool

		Gensym func(prefix string) *Symbol

		FindNamespace func(sym *lang.Symbol) *lang.Namespace
	}
)

// Analyze performs semantic analysis on the given s-expression,
// returning an AST.
func (a *Analyzer) Analyze(form interface{}, env Env) (*ast.Node2, error) {
	// defer func() {
	// 	fmt.Println("DONE Analyze", lang.ToString(form))
	// }()
	// fmt.Println("Analyze", lang.ToString(form))
	return a.analyzeForm(form, ctxEnv(env, ctxExpr).Assoc(KWTopLevel, true).(Env))
}

func (a *Analyzer) analyzeForm(form interface{}, env Env) (n *ast.Node2, err error) {
	switch v := form.(type) {
	case *Symbol:
		return a.analyzeSymbol(v, env)
	case IPersistentVector:
		return a.analyzeVector(v, env)
	case IPersistentMap:
		return a.analyzeMap(v, env)
	case IPersistentSet:
		return a.analyzeSet(v, env)
	case ISeq:
		return a.analyzeSeq(v, env)
	default:
		return a.analyzeConst(v, env)
	}
}

// analyzeSymbol performs semantic analysis on the given symbol,
// returning an AST.
func (a *Analyzer) analyzeSymbol(form *Symbol, env Env) (*ast.Node2, error) {
	mform, err := a.Macroexpand1(form)
	if err != nil {
		return nil, err
	}
	if !Equal(form, mform) {
		n, err := a.analyzeForm(mform, env)
		if err != nil {
			return nil, err
		}
		return withRawForm(n, form), nil
	}

	n := &ast.Node2{
		Env:  env,
		Form: mform,
	}
	if localBinding := Get(Get(env, KWLocals), form); localBinding != nil {
		// TODO: make sure this is correct...
		binding := localBinding.(*ast.Node2)
		bindingNode := binding.Sub.(*ast.BindingNode)

		n.Op = ast.OpLocal
		n.IsAssignable = binding.IsAssignable
		n.Sub = &ast.LocalNode{
			Name:       bindingNode.Name,
			Local:      bindingNode.Local,
			ArgID:      bindingNode.ArgID,
			IsVariadic: bindingNode.IsVariadic,
		}
	} else {
		v := a.resolveSym(form, env)
		vr, ok := v.(*Var)
		if ok {
			m := vr.Meta()
			n.Op = ast.OpVar
			n.Sub = &ast.VarNode{
				Var:  vr,
				Meta: m,
			}
			// IsAssignable: dynamicVar(vr, m), // TODO
		} else if v != nil {
			// The symbol resolves to a non-var  Treat it as a
			// constant.
			n.Op = ast.OpConst
			n.Sub = &ast.ConstNode{
				Type:  classifyType(v),
				Value: v,
			}
		} else {
			maybeClass := form.Namespace()
			if maybeClass != "" {
				n.Op = ast.OpMaybeHostForm // TODO: define this for Go interop
				n.Sub = &ast.MaybeHostFormNode{
					Class: maybeClass,
					Field: NewSymbol(form.Name()),
				}
			} else {
				n.Op = ast.OpMaybeClass
				n.Sub = &ast.MaybeClassNode{Class: mform}
			}
		}
	}

	return n, nil
}

// analyzeVector performs semantic analysis on the given vector,
// returning an AST.
func (a *Analyzer) analyzeVector(form IPersistentVector, env Env) (*ast.Node2, error) {
	n := ast.MakeNode2(ast.OpVector, form)
	var items []*ast.Node2
	for i := 0; i < form.Count(); i++ {
		// TODO: pass an "items-env" with an expr context
		nn, err := a.analyzeForm(MustNth(form, i), env)
		if err != nil {
			return nil, err
		}

		items = append(items, nn)
	}
	n.Sub = &ast.VectorNode{
		Items: items,
	}
	return n, nil
}

// analyzeMap performs semantic analysis on the given map,
// returning an AST.
func (a *Analyzer) analyzeMap(v IPersistentMap, env Env) (*ast.Node2, error) {
	n := ast.MakeNode2(ast.OpMap, v)
	var keys []*ast.Node2
	var vals []*ast.Node2
	for seq := Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass a "kv-env" with an expr context

		entry := seq.First().(*MapEntry)
		keyNode, err := a.analyzeForm(entry.Key(), env)
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
	n.Sub = &ast.MapNode{
		Keys: keys,
		Vals: vals,
	}
	return n, nil
}

// analyzeSet performs semantic analysis on the given set,
// returning an AST.
func (a *Analyzer) analyzeSet(v IPersistentSet, env Env) (*ast.Node2, error) {
	n := ast.MakeNode2(ast.OpSet, v)
	items := make([]*ast.Node2, 0, v.Count())
	for seq := Seq(v); seq != nil; seq = seq.Next() {
		// TODO: pass an "items-env" with an expr context
		item, err := a.analyzeForm(seq.First(), env)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	n.Sub = &ast.SetNode{
		Items: items,
	}
	return n, nil
}

// analyzeSeq performs semantic analysis on the given sequence,
// returning an AST.
func (a *Analyzer) analyzeSeq(form ISeq, env Env) (*ast.Node2, error) {
	if Seq(form) == nil {
		return a.analyzeConst(form, env)
	}

	op := form.First()
	if op == nil {
		return nil, exInfo("can't call nil", nil) // TODO: include form and source info
	}
	mform, err := a.Macroexpand1(form)
	if err != nil {
		return nil, err
	}

	if Equal(form, mform) {
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
func (a *Analyzer) analyzeConst(v interface{}, env Env) (*ast.Node2, error) {
	n := ast.MakeNode2(ast.OpConst, v)
	n.IsLiteral = true
	constNode := &ast.ConstNode{
		Type:  classifyType(v),
		Value: v,
	}
	n.Sub = constNode

	if im, ok := v.(IMeta); ok {
		meta := im.Meta()
		if meta != nil {
			mn, err := a.analyzeConst(meta, env)
			if err != nil {
				return nil, err
			}
			constNode.Meta = mn
		}
	}
	return n, nil
}

func (a *Analyzer) analyzeBody(body interface{}, env Env) (*ast.Node2, error) {
	n, err := a.parse(NewCons(NewSymbol("do"), body), env)
	if err != nil {
		return nil, err
	}
	n.Sub.(*ast.DoNode).IsBody = true
	return n, nil
}

// (defn analyze-let
//
//	[[op bindings & body :as form] {:keys [context loop-id] :as env}]
//	(validate-bindings form env)
//	(let [loop? (= 'loop* op)]
//	  (loop [bindings bindings
//	         env (ctx env :ctx/expr)
//	         binds []]
//	    (if-let [[name init & bindings] (seq bindings)]
//	      (if (not (valid-binding-symbol? name))
//	        (throw (ex-info (str "Bad binding form: " name)
//	                        (merge {:form form
//	                                :sym  name}
//	                               (-source-info form env))))
//	        (let [init-expr (analyze-form init env)
//	              bind-expr {:op       :binding
//	                         :env      env
//	                         :name     name
//	                         :init     init-expr
//	                         :form     name
//	                         :local    (if loop? :loop :let)
//	                         :children [:init]}]
//	          (recur bindings
//	                 (assoc-in env [:locals name] (dissoc-env bind-expr))
//	                 (conj binds bind-expr))))
//	      (let [body-env (assoc env :context (if loop? :ctx/return context))
//	            body (analyze-body body (merge body-env
//	                                           (when loop?
//	                                             {:loop-id     loop-id
//	                                              :loop-locals (count binds)})))]
//	        {:body     body
//	         :bindings binds
//	         :children [:bindings :body]})))))
func (a *Analyzer) analyzeLet(form interface{}, env Env) (*ast.Node2, error) {
	if Count(form) < 2 {
		return nil, exInfo("let requires a binding vector and a body", nil)
	}
	op := MustNth(form, 0)
	bindings := MustNth(form, 1)
	body := Rest(Rest(form))

	ctx := Get(env, KWContext)
	loopID := Get(env, KWLoopId)

	if err := a.validateBindings(form, env); err != nil {
		return nil, err
	}

	isLoop := Equal(op, symLoopStar)
	localKW := KWLet
	if isLoop {
		localKW = KWLoop
	}
	env = ctxEnv(env, KWCtxExpr)
	var binds []*ast.Node2
	for {
		bindingsSeq := Seq(bindings)
		if bindingsSeq == nil {
			break
		}
		name := bindingsSeq.First()
		init := second(bindingsSeq)
		bindings = Rest(Rest(bindingsSeq))
		if !isValidBindingSymbol(name) {
			return nil, exInfo("bad binding form: "+ToString(name), nil)
		}
		initExpr, err := a.analyzeForm(init, env)
		if err != nil {
			return nil, err
		}
		bindExpr := ast.MakeNode2(ast.OpBinding, name)
		bindExpr.Env = env
		bindExpr.Sub = &ast.BindingNode{
			Name:  name.(*lang.Symbol),
			Init:  initExpr,
			Local: localKW,
		}
		env = assocIn(env, vec(KWLocals, name), dissocEnv(bindExpr)).(Env)
		binds = append(binds, bindExpr)
	}
	if isLoop {
		ctx = KWCtxReturn
	}
	bodyEnv := Assoc(env, KWContext, ctx).(Env)
	if isLoop {
		bodyEnv = merge(bodyEnv, NewMap(
			KWLoopId, loopID,
			KWLoopLocals, Count(binds),
		)).(Env)
	}
	bodyExpr, err := a.analyzeBody(body, bodyEnv)
	if err != nil {
		return nil, err
	}
	return &ast.Node2{
		Sub: &ast.LetNode{
			Body:     bodyExpr,
			Bindings: binds,
		},
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Parse

func (a *Analyzer) parse(form interface{}, env Env) (*ast.Node2, error) {
	op := First(form)
	opSym, ok := op.(*Symbol)
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
	case "case*":
		return a.parseCaseStar(form, env)
	}

	return a.parseInvoke(form, env)
}

func (a *Analyzer) parseInvoke(form interface{}, env Env) (*ast.Node2, error) {
	f := First(form)
	args := Rest(form)
	fenv := ctxEnv(env, ctxExpr)
	fnExpr, err := a.analyzeForm(f, fenv)
	if err != nil {
		return nil, err
	}
	argsExprs := make([]*ast.Node2, 0, Count(args))
	for seq := Seq(args); seq != nil; seq = seq.Next() {
		n, err := a.analyzeForm(seq.First(), fenv)
		if err != nil {
			return nil, err
		}
		argsExprs = append(argsExprs, n)
	}
	var meta IPersistentMap
	if m, ok := form.(IMeta); ok && Seq(m.Meta()) != nil {
		meta = m.Meta()
	}
	n := ast.MakeNode2(ast.OpInvoke, form)
	n.Sub = &ast.InvokeNode{
		Meta: meta,
		Fn:   fnExpr,
		Args: argsExprs,
	}
	return n, nil
}

// (defn parse-do
//
//	[[_ & exprs :as form] env]
//	(let [statements-env (ctx env :ctx/statement)
//	      [statements ret] (loop [statements [] [e & exprs] exprs]
//	                         (if (seq exprs)
//	                           (recur (conj statements e) exprs)
//	                           [statements e]))
//	      statements (mapv (analyze-in-env statements-env) statements)
//	      ret (analyze-form ret env)]
//	  {:op         :do
//	   :env        env
//	   :form       form
//	   :statements statements
//	   :ret        ret
//	   :children   [:statements :ret]}))
func (a *Analyzer) parseDo(form interface{}, env Env) (*ast.Node2, error) {
	exprs := Rest(form)
	statementsEnv := ctxEnv(env, ctxStatement)
	var statementForms []interface{}
	retForm := First(exprs)
	for exprs := Seq(Rest(exprs)); exprs != nil; exprs = exprs.Next() {
		statementForms = append(statementForms, retForm)
		retForm = exprs.First()
	}
	statements := make([]*ast.Node2, len(statementForms))
	for i, form := range statementForms {
		s, err := a.analyzeForm(form, statementsEnv)
		if err != nil {
			return nil, err
		}
		statements[i] = s
	}

	ret, err := a.analyzeForm(retForm, env)
	if err != nil {
		return nil, err
	}

	n := ast.MakeNode2(ast.OpDo, form)
	n.Env = env
	n.Sub = &ast.DoNode{
		Statements: statements,
		Ret:        ret,
	}
	return n, nil
}

// (defn parse-if
//
//	[[_ test then else :as form] env]
//	(let [formc (count form)]
//	  (when-not (or (= formc 3) (= formc 4))
//	    (throw (ex-info (str "Wrong number of args to if, had: " (dec (count form)))
//	                    (merge {:form form}
//	                           (-source-info form env))))))
//	(let [test-expr (analyze-form test (ctx env :ctx/expr))
//	      then-expr (analyze-form then env)
//	      else-expr (analyze-form else env)]
//	  {:op       :if
//	   :form     form
//	   :env      env
//	   :test     test-expr
//	   :then     then-expr
//	   :else     else-expr
//	   :children [:test :then :else]}))
func (a *Analyzer) parseIf(form interface{}, env Env) (*ast.Node2, error) {
	formc := Count(form)
	if formc != 3 && formc != 4 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to if, had: %d", formc-1), nil)
	}

	test := MustNth(form, 1)
	then := MustNth(form, 2)
	var els interface{}
	if formc == 4 {
		els = MustNth(form, 3)
	}
	testExpr, err := a.analyzeForm(test, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	thenExpr, err := a.analyzeForm(then, env)
	if err != nil {
		return nil, err
	}
	elseExpr, err := a.analyzeForm(els, env)
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpIf, form)
	n.Env = env
	n.Sub = &ast.IfNode{
		Test: testExpr,
		Then: thenExpr,
		Else: elseExpr,
	}
	return n, nil
}

// (defn parse-new
//
//	[[_ class & args :as form] env]
//	(when-not (>= (count form) 2)
//	  (throw (ex-info (str "Wrong number of args to new, had: " (dec (count form)))
//	                  (merge {:form form}
//	                         (-source-info form env)))))
//	(let [args-env (ctx env :ctx/expr)
//	      args (mapv (analyze-in-env args-env) args)]
//	  {:op          :new
//	   :env         env
//	   :form        form
//	   :class       (analyze-form class (assoc env :locals {})) ;; avoid shadowing
//	   :args        args
//	   :children    [:class :args]}))
func (a *Analyzer) parseNew(form interface{}, env Env) (*ast.Node2, error) {
	if Count(form) < 2 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to new, had: %d", Count(form)-1), nil)
	}

	class := MustNth(form, 1)
	args := Rest(Rest(form))
	argsEnv := ctxEnv(env, ctxExpr)
	var argsExprs []*ast.Node2
	for seq := Seq(args); seq != nil; seq = seq.Next() {
		arg, err := a.analyzeForm(seq.First(), argsEnv)
		if err != nil {
			return nil, err
		}
		argsExprs = append(argsExprs, arg)
	}
	classExpr, err := a.analyzeForm(class, env.Assoc(KWLocals, NewMap()).(Env))
	if err != nil {
		return nil, err
	}

	n := ast.MakeNode2(ast.OpNew, form)
	n.Env = env
	n.Sub = &ast.NewNode{
		Class: classExpr,
		Args:  argsExprs,
	}
	return n, nil
}

func (a *Analyzer) parseQuote(form interface{}, env Env) (*ast.Node2, error) {
	expr := second(form)
	if Count(form) != 2 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to quote, had: %v", Count(form)-1), nil)
	}
	cnst, err := a.analyzeConst(expr, env)
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpQuote, form)
	n.Env = env
	n.IsLiteral = true
	n.Sub = &ast.QuoteNode{
		Expr: cnst,
	}
	return n, nil
}

// (defn parse-set!
//
//	[[_ target val :as form] env]
//	(when-not (= 3 (count form))
//	  (throw (ex-info (str "Wrong number of args to set!, had: " (dec (count form)))
//	                  (merge {:form form}
//	                         (-source-info form env)))))
//	(let [target (analyze-form target (ctx env :ctx/expr))
//	      val (analyze-form val (ctx env :ctx/expr))]
//	  {:op       :set!
//	   :env      env
//	   :form     form
//	   :target   target
//	   :val      val
//	   :children [:target :val]}))
func (a *Analyzer) parseSetBang(form interface{}, env Env) (*ast.Node2, error) {
	if Count(form) != 3 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to set!, had: %d", Count(form)-1), nil)
	}
	target := MustNth(form, 1)
	val := MustNth(form, 2)

	targetExpr, err := a.analyzeForm(target, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	valExpr, err := a.analyzeForm(val, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpSetBang, form)
	n.Env = env
	n.Sub = &ast.SetBangNode{
		Target: targetExpr,
		Val:    valExpr,
	}
	return n, nil
}

// (defn parse-try
//
//	[[_ & body :as form] env]
//	(let [catch? (every-pred seq? #(= (first %) 'catch))
//	      finally? (every-pred seq? #(= (first %) 'finally))
//	      [body tail'] (split-with' (complement (some-fn catch? finally?)) body)
//	      [cblocks tail] (split-with' catch? tail')
//	      [[fblock & fbs :as fblocks] tail] (split-with' finally? tail)]
//	  (when-not (empty? tail)
//	    (throw (ex-info "Only catch or finally clause can follow catch in try expression"
//	                    (merge {:expr tail
//	                            :form form}
//	                           (-source-info form env)))))
//	  (when-not (empty? fbs)
//	    (throw (ex-info "Only one finally clause allowed in try expression"
//	                    (merge {:expr fblocks
//	                            :form form}
//	                           (-source-info form env)))))
//	  (let [env' (assoc env :in-try true)
//	        body (analyze-body body env')
//	        cenv (ctx env' :ctx/expr)
//	        cblocks (mapv #(parse-catch % cenv) cblocks)
//	        fblock (when-not (empty? fblock)
//	                 (analyze-body (rest fblock) (ctx env :ctx/statement)))]
//	    (merge {:op      :try
//	            :env     env
//	            :form    form
//	            :body    body
//	            :catches cblocks}
//	           (when fblock
//	             {:finally fblock})
//	           {:children (into [:body :catches]
//	                            (when fblock [:finally]))}))))
func (a *Analyzer) parseTry(form interface{}, env Env) (*ast.Node2, error) {
	catch := func(form interface{}) bool {
		return IsSeq(form) && Equal(symCatch, First(form))
	}
	finally := func(form interface{}) bool {
		return IsSeq(form) && Equal(symFinally, First(form))
	}
	body, tail := splitWith(func(form interface{}) bool {
		return !catch(form) && !finally(form)
	}, Rest(form))
	cblocks, tail := splitWith(catch, tail)
	fblocks, tail := splitWith(finally, tail)
	if Count(tail) != 0 {
		return nil, exInfo("only catch or finally clause can follow catch in try expression", nil)
	}
	if Count(fblocks) > 1 {
		return nil, exInfo("only one finally clause allowed in try expression", nil)
	}
	env = env.Assoc(KWInTry, true).(Env)
	bodyExpr, err := a.analyzeBody(body, env)
	if err != nil {
		return nil, err
	}
	cenv := ctxEnv(env, ctxExpr)
	var cblocksExpr []*ast.Node2
	for cblockSeq := Seq(cblocks); cblockSeq != nil; cblockSeq = Next(cblockSeq) {
		cblock := First(cblockSeq)
		cblockExpr, err := a.parseCatch(cblock, cenv)
		if err != nil {
			return nil, err
		}
		cblocksExpr = append(cblocksExpr, cblockExpr)
	}
	var fblockExpr *ast.Node2
	if Count(fblocks) > 0 {
		fblock := First(fblocks)
		fblockExpr, err = a.analyzeBody(Rest(fblock), ctxEnv(env, ctxStatement))
		if err != nil {
			return nil, err
		}
	}
	children := []interface{}{KWBody, KWCatches}
	if fblockExpr != nil {
		children = append(children, KWFinally)
	}
	n := ast.MakeNode2(ast.OpTry, form)
	n.Env = env
	n.Sub = &ast.TryNode{
		Body:    bodyExpr,
		Catches: cblocksExpr,
		Finally: fblockExpr,
	}
	return n, nil
}

// (defn parse-catch
//
//	[[_ etype ename & body :as form] env]
//	(when-not (valid-binding-symbol? ename)
//	  (throw (ex-info (str "Bad binding form: " ename)
//	                  (merge {:sym ename
//	                          :form form}
//	                         (-source-info form env)))))
//	(let [env (dissoc env :in-try)
//	      local {:op    :binding
//	             :env   env
//	             :form  ename
//	             :name  ename
//	             :local :catch}]
//	  {:op          :catch
//	   :class       (analyze-form etype (assoc env :locals {}))
//	   :local       local
//	   :env         env
//	   :form        form
//	   :body        (analyze-body body (assoc-in env [:locals ename] (dissoc-env local)))
//	   :children    [:class :local :body]}))
func (a *Analyzer) parseCatch(form interface{}, env Env) (*ast.Node2, error) {
	etype := First(Rest(form))
	ename := First(Rest(Rest(form)))
	if !isValidBindingSymbol(ename) {
		return nil, exInfo("bad binding form: "+ToString(ename), nil)
	}
	env = Dissoc(env, KWInTry).(Env)

	local := ast.MakeNode2(ast.OpBinding, ename)
	local.Env = env
	local.Sub = &ast.BindingNode{
		Name:  ename.(*lang.Symbol),
		Local: KWCatch,
	}

	body, err := a.analyzeBody(Rest(Rest(Rest(form))), env.Assoc(KWLocals, NewMap()).(Env))
	if err != nil {
		return nil, err
	}
	class, err := a.analyzeForm(etype, env.Assoc(KWLocals, NewMap()).(Env))
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpCatch, form)
	n.Env = env
	n.Sub = &ast.CatchNode{
		Class: class,
		Local: local,
		Body:  body,
	}
	return n, nil
}

// (defn parse-throw
//
//	[[_ throw :as form] env]
//	(when-not (= 2 (count form))
//	  (throw (ex-info (str "Wrong number of args to throw, had: " (dec (count form)))
//	                  (merge {:form form}
//	                         (-source-info form env)))))
//	{:op        :throw
//	 :env       env
//	 :form      form
//	 :exception (analyze-form throw (ctx env :ctx/expr))
//	 :children  [:exception]})
func (a *Analyzer) parseThrow(form interface{}, env Env) (*ast.Node2, error) {
	throw := second(form)
	if Count(form) != 2 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to throw, had: %d", Count(form)-1), nil)
	}
	exception, err := a.analyzeForm(throw, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpThrow, form)
	n.Env = env
	n.Sub = &ast.ThrowNode{
		Exception: exception,
	}
	return n, nil
}

func (a *Analyzer) parseDef(form interface{}, env Env) (*ast.Node2, error) {
	symForm := First(Rest(form))
	expr := Rest(Rest(form))

	sym, ok := symForm.(*Symbol)
	if !ok {
		return nil, exInfo(fmt.Sprintf("first argument to def must be a symbol, got %T", symForm), nil)
	}

	if sym.Namespace() != "" && sym.Namespace() != Get(env, KWNS).(*Symbol).Name() {
		return nil, exInfo("can't def namespace-qualified symbol", nil)
	}

	var init, doc interface{}
	hasInit := true
	switch Count(expr) {
	case 0:
		hasInit = false
	case 1:
		init = First(expr)
	case 2:
		doc = First(expr)
		init = First(Rest(expr))
	default:
		return nil, exInfo("invalid def", nil)
	}
	if doc == nil {
		doc = Get(sym.Meta(), KWDoc)
	} else if _, ok := doc.(string); !ok {
		return nil, exInfo("doc must be a string", nil)
	}
	arglists := Get(sym.Meta(), KWArglists)
	if arglists != nil {
		arglists = second(arglists)
	}
	sym = NewSymbol(sym.Name()).WithMeta(
		merge(NewMap(), // hack to make sure we get a non-nil map
			sym.Meta(),
			mapWhen(KWArglists, arglists),
			mapWhen(KWDoc, doc),
			// TODO: source info
		).(IPersistentMap)).(*Symbol)

	vr, err := a.CreateVar(sym, env)
	if err != nil {
		return nil, err
	}
	// TODO: make sure we have an environment
	// TODO: mutate the environment to add the var to the namespace

	meta := sym.Meta()
	if arglists != nil {
		meta = merge(meta, NewMap(KWArglists, NewList(NewSymbol("quote"), arglists))).(IPersistentMap)
	}
	var metaExpr *ast.Node2
	if meta != nil {
		var err error
		metaExpr, err = a.analyzeForm(meta, ctxEnv(env, ctxExpr))
		if err != nil {
			return nil, err
		}
	}

	var initNode *ast.Node2
	if hasInit {
		initNode, err = a.analyzeForm(init, ctxEnv(env, ctxExpr))
		if err != nil {
			return nil, err
		}
	}

	n := ast.MakeNode2(ast.OpDef, form)
	n.Env = env
	n.Sub = &ast.DefNode{
		Name: sym,
		Var:  vr.(*lang.Var),
		Meta: metaExpr,
		Init: initNode,
		Doc:  doc,
	}
	return n, nil
}

// (defn parse-dot
//
//	[[_ target & [m-or-f & args] :as form] env]
//	(when-not (>= (count form) 3)
//	  (throw (ex-info (str "Wrong number of args to ., had: " (dec (count form)))
//	                  (merge {:form form}
//	                         (-source-info form env)))))
//	(let [[m-or-f field?] (if (and (symbol? m-or-f)
//	                               (= \- (first (name m-or-f))))
//	                        [(-> m-or-f name (subs 1) symbol) true]
//	                        [(if args (cons m-or-f args) m-or-f) false])
//	      target-expr (analyze-form target (ctx env :ctx/expr))
//	      call? (and (not field?) (seq? m-or-f))]
//	  (when (and call? (not (symbol? (first m-or-f))))
//	    (throw (ex-info (str "Method name must be a symbol, had: " (class (first m-or-f)))
//	                    (merge {:form   form
//	                            :method m-or-f}
//	                           (-source-info form env)))))
//	  (merge {:form   form
//	          :env    env
//	          :target target-expr}
//	         (cond
//	          call?
//	          {:op       :host-call
//	           :method   (symbol (name (first m-or-f)))
//	           :args     (mapv (analyze-in-env (ctx env :ctx/expr)) (next m-or-f))
//	           :children [:target :args]}
//	          field?
//	          {:op          :host-field
//	           :assignable? true
//	           :field       (symbol (name m-or-f))
//	           :children    [:target]}
//	          :else
//	          {:op          :host-interop ;; either field access or no-args method call
//	           :assignable? true
//	           :m-or-f      (symbol (name m-or-f))
//	           :children    [:target]}))))
func (a *Analyzer) parseDot(form interface{}, env Env) (*ast.Node2, error) {
	if Count(form) < 3 {
		return nil, exInfo("wrong number of args to ., had: %d", Count(form)-1)
	}
	target := second(form)
	mOrF := MustNth(form, 2)
	args := Rest(Rest(Rest(form)))
	isField := false
	if sym, ok := mOrF.(*Symbol); ok && len(sym.Name()) > 0 && sym.Name()[0] == '-' {
		mOrF = NewSymbol(sym.Name()[1:])
		isField = true
	} else if Count(args) != 0 {
		mOrF = NewCons(mOrF, args)
	}
	targetExpr, err := a.analyzeForm(target, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	call := false
	if _, ok := mOrF.(ISeq); ok && !isField {
		call = true
	}
	if call {
		if _, ok := First(mOrF).(*Symbol); !ok {
			return nil, exInfo(fmt.Sprintf("method name must be a symbol, had: %T", First(mOrF)), nil)
		}
	}

	switch {
	case call:
		var argNodes []*ast.Node2
		for seq := Seq(Rest(mOrF)); seq != nil; seq = seq.Next() {
			arg := First(seq)
			argNode, err := a.analyzeForm(arg, ctxEnv(env, ctxExpr))
			if err != nil {
				return nil, err
			}
			argNodes = append(argNodes, argNode)
		}

		n := ast.MakeNode2(ast.OpHostCall, form)
		n.Env = env
		n.Sub = &ast.HostCallNode{
			Target: targetExpr,
			Method: NewSymbol(First(mOrF).(*Symbol).Name()),
			Args:   argNodes,
		}
		return n, nil
	case isField:
		n := ast.MakeNode2(ast.OpHostField, form)
		n.Env = env
		n.IsAssignable = true
		n.Sub = &ast.HostFieldNode{
			Target: targetExpr,
			Field:  NewSymbol(mOrF.(*Symbol).Name()),
		}
		return n, nil
	default:
		n := ast.MakeNode2(ast.OpHostInterop, form)
		n.Env = env
		n.IsAssignable = true
		n.Sub = &ast.HostInteropNode{
			Target: targetExpr,
			MOrF:   NewSymbol(mOrF.(*Symbol).Name()),
		}
		return n, nil
	}
}

// (defn parse-let*
//
//	[form env]
//	(into {:op   :let
//	       :form form
//	       :env  env}
//	      (analyze-let form env)))
func (a *Analyzer) parseLetStar(form interface{}, env Env) (*ast.Node2, error) {
	let, err := a.analyzeLet(form, env)
	if err != nil {
		return nil, err
	}
	if let.Op == ast.OpUnknown {
		let.Op = ast.OpLet
	}
	if let.Form == nil {
		let.Form = form
	}
	if let.Env == nil {
		let.Env = env
	}
	return let, nil
}

// (defn parse-letfn*
//
//	[[_ bindings & body :as form] env]
//	(validate-bindings form env)
//	(let [bindings (apply array-map bindings) ;; pick only one local with the same name, if more are present.
//	      fns      (keys bindings)]
//	  (when-let [[sym] (seq (remove valid-binding-symbol? fns))]
//	    (throw (ex-info (str "Bad binding form: " sym)
//	                    (merge {:form form
//	                            :sym  sym}
//	                           (-source-info form env)))))
//	  (let [binds (reduce (fn [binds name]
//	                        (assoc binds name
//	                               {:op    :binding
//	                                :env   env
//	                                :name  name
//	                                :form  name
//	                                :local :letfn}))
//	                      {} fns)
//	        e (update-in env [:locals] merge binds) ;; pre-seed locals
//	        binds (reduce-kv (fn [binds name bind]
//	                           (assoc binds name
//	                                  (merge bind
//	                                         {:init     (analyze-form (bindings name)
//	                                                                  (ctx e :ctx/expr))
//	                                          :children [:init]})))
//	                         {} binds)
//	        e (update-in env [:locals] merge (update-vals binds dissoc-env))
//	        body (analyze-body body e)]
//	    {:op       :letfn
//	     :env      env
//	     :form     form
//	     :bindings (vec (vals binds)) ;; order is irrelevant
//	     :body     body
//	     :children [:bindings :body]})))
func (a *Analyzer) parseLetfnStar(form interface{}, env Env) (*ast.Node2, error) {
	if Count(form) < 2 {
		return nil, exInfo("letfn requires a binding vector and a body", nil)
	}
	bindings := second(form)
	body := Rest(Rest(form))
	if err := a.validateBindings(form, env); err != nil {
		return nil, err
	}
	bindingsMap := NewMap().(Associative)
	for seq := Seq(bindings); seq != nil; seq = seq.Next().Next() {
		bindingsMap = bindingsMap.Assoc(First(seq), second(seq))
	}
	fns := Keys(bindingsMap)
	if sym := First(Seq(removeP(isValidBindingSymbol, fns))); sym != nil {
		return nil, exInfo("bad binding form: "+ToString(sym), nil)
	}
	bindsMap := map[*lang.Symbol]*ast.Node2{}
	for s := Seq(fns); s != nil; s = s.Next() {
		name := s.First().(*lang.Symbol)
		bind := ast.MakeNode2(ast.OpBinding, name)
		bind.Env = env
		bind.Sub = &ast.BindingNode{
			Name:  name,
			Local: KWLetfn,
		}
		bindsMap[name] = bind
	}
	e := env
	for name, bind := range bindsMap {
		e = updateIn(env, vec(KWLocals), merge, NewMap(name, bind)).(Env)
	}
	var binds []*ast.Node2
	for name, bind := range bindsMap {
		init, err := a.analyzeForm(Get(bindingsMap, name), ctxEnv(e, ctxExpr))
		if err != nil {
			panic(err)
		}
		bind.Sub.(*ast.BindingNode).Init = init
		binds = append(binds, bind)
	}

	for name, bind := range bindsMap {
		e = updateIn(e, vec(KWLocals), merge, NewMap(name, dissocEnv(bind))).(Env)
	}

	bodyExpr, err := a.analyzeBody(body, e)
	if err != nil {
		return nil, err
	}
	n := ast.MakeNode2(ast.OpLetFn, form)
	n.Env = env
	n.Sub = &ast.LetFnNode{
		Bindings: binds,
		Body:     bodyExpr,
	}
	return n, nil
}

// (defn parse-loop*
//
//	[form env]
//	(let [loop-id (gensym "loop_") ;; can be used to find matching recur
//	      env (assoc env :loop-id loop-id)]
//	  (into {:op      :loop
//	         :form    form
//	         :env     env
//	         :loop-id loop-id}
//	        (analyze-let form env))))
func (a *Analyzer) parseLoopStar(form interface{}, env Env) (*ast.Node2, error) {
	loopID := a.Gensym("loop_")
	env = env.Assoc(KWLoopId, loopID).(Env)
	loop, err := a.analyzeLet(form, env)
	if err != nil {
		return nil, err
	}
	if loop.Op == ast.OpUnknown {
		loop.Op = ast.OpLoop
	}
	if loop.Form == nil {
		loop.Form = form
	}
	if loop.Env == nil {
		loop.Env = env
	}
	loop.Sub.(*ast.LetNode).LoopID = loopID

	return loop, nil
}

// (defn parse-recur
//
//	[[_ & exprs :as form] {:keys [context loop-locals loop-id]
//	                       :as env}]
//	(when-let [error-msg
//	           (cond
//	            (not (isa? context :ctx/return))
//	            "Can only recur from tail position"
//	            (not (= (count exprs) loop-locals))
//	            (str "Mismatched argument count to recur, expected: " loop-locals
//	                 " args, had: " (count exprs)))]
//	  (throw (ex-info error-msg
//	                  (merge {:exprs exprs
//	                          :form  form}
//	                         (-source-info form env)))))
//	(let [exprs (mapv (analyze-in-env (ctx env :ctx/expr)) exprs)]
//	  {:op          :recur
//	   :env         env
//	   :form        form
//	   :exprs       exprs
//	   :loop-id     loop-id
//	   :children    [:exprs]}))
func (a *Analyzer) parseRecur(form interface{}, env Env) (*ast.Node2, error) {
	exprs := Rest(form)
	ctx := Get(env, KWContext)
	loopLocals := Get(env, KWLoopLocals)
	loopID := Get(env, KWLoopId)

	errorMsg := ""
	switch {
	case !Equal(ctx, ctxReturn):
		errorMsg = "can only recur from tail position"
	case !Equal(Count(exprs), loopLocals):
		errorMsg = fmt.Sprintf("mismatched argument count to recur, expected: %v args, had: %v", loopLocals, Count(exprs))
	}
	if errorMsg != "" {
		return nil, exInfo(errorMsg, nil)
	}
	var exprNodes []*ast.Node2
	for seq := Seq(exprs); seq != nil; seq = seq.Next() {
		expr := First(seq)
		exprNode, err := a.analyzeForm(expr, ctxEnv(env, ctxExpr))
		if err != nil {
			return nil, err
		}
		exprNodes = append(exprNodes, exprNode)
	}

	n := ast.MakeNode2(ast.OpRecur, form)
	n.Env = env
	n.Sub = &ast.RecurNode{
		Exprs:  exprNodes,
		LoopID: loopID.(*lang.Symbol),
	}
	return n, nil
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
func (a *Analyzer) parseFnStar(form interface{}, env Env) (*ast.Node2, error) {
	fnSym, _ := First(form).(*Symbol)
	args := Rest(form)

	var name *Symbol
	var meths interface{}
	if sym, ok := First(args).(*Symbol); ok {
		name = sym
		meths = Next(args)
	} else {
		meths = Seq(args)
	}
	nameExpr := ast.MakeNode2(ast.OpBinding, name)
	nameExpr.Env = env
	nameExpr.Sub = &ast.BindingNode{
		Name:  name,
		Local: KWFn,
	}
	e := env
	if name != nil {
		e = assocIn(env, vec(KWLocals, name), dissocEnv(nameExpr)).(Env)
		e = Assoc(e, KWLocal, nameExpr).(Env)
	}

	once := false
	if fnSym != nil {
		if o, ok := Get(fnSym.Meta(), KWOnce).(bool); ok && o {
			once = true
		}
	}
	menv := Assoc(Dissoc(e, KWInTry), KWOnce, once)
	if _, ok := First(meths).(*Vector); ok {
		meths = NewList(meths)
	}
	var methodsExprs []*ast.Node2
	for seq := Seq(meths); seq != nil; seq = seq.Next() {
		m, err := a.analyzeFnMethod(First(seq), menv.(Env))
		if err != nil {
			return nil, err
		}
		methodsExprs = append(methodsExprs, m)
	}
	variadic := false
	for _, m := range methodsExprs {
		if m.Sub.(*ast.FnMethodNode).IsVariadic {
			variadic = true
			break
		}
	}
	maxFixedArity := 0
	arities := make(map[int]bool)
	sawVariadic := false
	variadicArity := 0
	for _, m := range methodsExprs {
		fnMethodNode := m.Sub.(*ast.FnMethodNode)
		arity := fnMethodNode.FixedArity
		if fnMethodNode.IsVariadic {
			if sawVariadic {
				return nil, errors.New("can't have more than 1 variadic overload")
			}
			sawVariadic = true
			variadicArity = arity
		} else if _, ok := arities[arity]; ok {
			return nil, exInfo("can't have 2 or more overloads with the same arity", nil)
		}
		arities[arity] = true
		if arity > maxFixedArity {
			maxFixedArity = arity
		}
	}
	if sawVariadic && maxFixedArity > variadicArity {
		return nil, exInfo("can't have fixed arity overload with more params than variadic overload", nil)
	}

	n := ast.MakeNode2(ast.OpFn, form)
	n.Env = env
	fnNode := &ast.FnNode{
		IsVariadic:    variadic,
		MaxFixedArity: maxFixedArity,
		Methods:       methodsExprs,
		Once:          once,
	}
	n.Sub = fnNode
	if name != nil {
		fnNode.Local = nameExpr
	}

	return a.wrappingMeta(n)
}

// {:op   :case
//
//	:doc  "Node for a case* special-form expression"
//	:keys [[:form "`(case* expr shift mask default case-map switch-type test-type skip-check?)`"]
//	       ^:children
//	       [:test "The AST node for the expression to test against"]
//	       ^:children
//	       [:nodes "A vector of :case-node AST nodes representing the test/then clauses of the case* expression"]
//	       ^:children
//	       [:default "An AST node representing the default value of the case expression"]
//	       ]}
func (a *Analyzer) parseCaseStar(form interface{}, env Env) (*ast.Node2, error) {
	var expr *Symbol
	var shift, mask int64
	var defaultForm interface{}
	var caseMap IPersistentMap
	var switchType, testType Keyword
	var skipCheck interface{}
	var err error
	if Count(form) == 9 {
		err = unpackSeq(Rest(form), &expr, &shift, &mask, &defaultForm, &caseMap, &switchType, &testType, &skipCheck)
	} else {
		err = unpackSeq(Rest(form), &expr, &shift, &mask, &defaultForm, &caseMap, &switchType, &testType)
	}
	if err != nil {
		return nil, exInfo(fmt.Sprintf("case*: %v", err), nil)
	}
	if switchType != KWCompact && switchType != KWSparse {
		return nil, exInfo(fmt.Sprintf("unexpected shift type: %v", switchType), nil)
	}
	if testType != KWInt && testType != KWHashIdentity && testType != KWHashEquiv {
		return nil, exInfo(fmt.Sprintf("unexpected test type: %v", testType), nil)
	}

	testExpr, err := a.analyzeForm(expr, ctxEnv(env, KWCtxExpr))
	if err != nil {
		return nil, err
	}
	defaultExpr, err := a.analyzeForm(defaultForm, env)
	if err != nil {
		return nil, err
	}

	var nodes []*ast.Node2
	for seq := Seq(caseMap); seq != nil; seq = seq.Next() {
		// TODO: is the shift, mask, etc. relevant for anything but
		// performance? omitting for now.
		entry := First(seq).(IMapEntry).Val()
		cond, then := First(entry), second(entry)
		// TODO: support a vector of conditions
		condExpr, err := a.analyzeConst(cond, ctxEnv(env, KWCtxExpr))
		if err != nil {
			return nil, err
		}
		thenExpr, err := a.analyzeForm(then, env)
		if err != nil {
			return nil, err
		}
		caseNode := ast.MakeNode2(ast.OpCaseNode, form)
		caseNode.Env = env
		caseNode.Sub = &ast.CaseNodeNode{
			Tests: []*ast.Node2{condExpr},
			Then:  thenExpr,
		}
		nodes = append(nodes, caseNode)
	}

	n := ast.MakeNode2(ast.OpCase, form)
	n.Env = env
	n.Sub = &ast.CaseNode{
		Test:    testExpr,
		Nodes:   nodes,
		Default: defaultExpr,
	}
	return n, nil
}

// (defn analyze-fn-method [[params & body :as form] {:keys [locals local] :as env}]
//
//	(when-not (vector? params)
//	  (throw (ex-info "Parameter declaration should be a vector"
//	                  (merge {:params params
//	                          :form   form}
//	                         (-source-info form env)
//	                         (-source-info params env)))))
//	(when (not-every? valid-binding-symbol? params)
//	  (throw (ex-info (str "Params must be valid binding symbols, had: "
//	                       (mapv class params))
//	                  (merge {:params params
//	                          :form   form}
//	                         (-source-info form env)
//	                         (-source-info params env))))) ;; more specific
//	(let [variadic? (boolean (some '#{&} params))
//	      params-names (if variadic? (conj (pop (pop params)) (peek params)) params)
//	      env (dissoc env :local)
//	      arity (count params-names)
//	      params-expr (mapv (fn [name id]
//	                          {:env       env
//	                           :form      name
//	                           :name      name
//	                           :variadic? (and variadic?
//	                                           (= id (dec arity)))
//	                           :op        :binding
//	                           :arg-id    id
//	                           :local     :arg})
//	                        params-names (range))
//	      fixed-arity (if variadic?
//	                    (dec arity)
//	                    arity)
//	      loop-id (gensym "loop_")
//	      body-env (into (update-in env [:locals]
//	                                merge (zipmap params-names (map dissoc-env params-expr)))
//	                     {:context     :ctx/return
//	                      :loop-id     loop-id
//	                      :loop-locals (count params-expr)})
//	      body (analyze-body body body-env)]
//	  (when variadic?
//	    (let [x (drop-while #(not= % '&) params)]
//	      (when (contains? #{nil '&} (second x))
//	        (throw (ex-info "Invalid parameter list"
//	                        (merge {:params params
//	                                :form   form}
//	                               (-source-info form env)
//	                               (-source-info params env)))))
//	      (when (not= 2 (count x))
//	        (throw (ex-info (str "Unexpected parameter: " (first (drop 2 x))
//	                             " after variadic parameter: " (second x))
//	                        (merge {:params params
//	                                :form   form}
//	                               (-source-info form env)
//	                               (-source-info params env)))))))
//	  (merge
//	   {:op          :fn-method
//	    :form        form
//	    :loop-id     loop-id
//	    :env         env
//	    :variadic?   variadic?
//	    :params      params-expr
//	    :fixed-arity fixed-arity
//	    :body        body
//	    :children    [:params :body]}
//	   (when local
//	     {:local (dissoc-env local)}))))
func (a *Analyzer) analyzeFnMethod(form interface{}, env Env) (*ast.Node2, error) {
	if _, ok := form.(ISeqable); !ok {
		return nil, exInfo("invalid fn method", nil)
	}
	params, ok := First(form).(IPersistentVector)
	if !ok {
		return nil, exInfo("parameter declaration should be a vector", nil)
	}
	body := Rest(form)

	var variadic bool
	var variadicParams ISeq
	for seq := Seq(params); seq != nil; seq = seq.Next() {
		if !isValidBindingSymbol(seq.First()) {
			return nil, exInfo(fmt.Sprintf("params must be valid binding symbols, had: %T", seq.First()), nil)
		}
		if seq.First().(*Symbol).Name() == "&" {
			if variadic {
				return nil, exInfo("can't have more than 1 variadic param", nil)
			}
			variadic = true
			variadicParams = seq.Next()
		}
	}
	paramsNames := params
	if variadic {
		if Count(variadicParams) != 1 {
			return nil, exInfo("variadic method must have exactly 1 param", nil)
		}
		paramsNames = params.Pop().Pop().(Conjer).Conj(params.Peek()).(IPersistentVector)
	}
	env = Dissoc(env, KWLocal).(Env)
	arity := paramsNames.Count()
	var paramsExpr []*ast.Node2
	id := 0
	for seq := Seq(paramsNames); seq != nil; seq, id = seq.Next(), id+1 {
		name := seq.First()
		param := ast.MakeNode2(ast.OpBinding, name)
		param.Env = env
		param.Sub = &ast.BindingNode{
			Name:       name.(*Symbol),
			ArgID:      id,
			Local:      KWArg,
			IsVariadic: variadic && id == arity-1,
		}
		paramsExpr = append(paramsExpr, param)
	}
	fixedArity := arity
	if variadic {
		fixedArity = arity - 1
	}
	loopID := a.Gensym("loop_")
	var bodyEnv Env
	{
		var localsMap IPersistentMap = NewMap()
		if locals, ok := Get(env, KWLocals).(IPersistentMap); ok {
			localsMap = locals
		}
		for i := 0; i < paramsNames.Count(); i++ {
			localsMap = localsMap.Assoc(MustNth(paramsNames, i), dissocEnv(paramsExpr[i])).(IPersistentMap)
		}
		bodyEnv = env.Assoc(KWLocals, localsMap).(Env)
	}
	bodyEnv = merge(bodyEnv,
		NewMap(
			KWContext, KWCtxReturn,
			KWLoopId, loopID,
			KWLoopLocals, len(paramsExpr),
		),
	).(Env)
	bodyNode, err := a.analyzeBody(body, bodyEnv)
	if err != nil {
		return nil, err
	}

	n := ast.MakeNode2(ast.OpFnMethod, form)
	n.Env = env
	n.Sub = &ast.FnMethodNode{
		Params:     paramsExpr,
		FixedArity: fixedArity,
		Body:       bodyNode,
		LoopID:     loopID, // TODO?
		IsVariadic: variadic,
	}
	return n, nil
}

// (defn parse-var
//
//	[[_ var :as form] env]
//	(when-not (= 2 (count form))
//	  (throw (ex-info (str "Wrong number of args to var, had: " (dec (count form)))
//	                  (merge {:form form}
//	                         (-source-info form env)))))
//	(if-let [var (resolve-sym var env)]
//	  {:op   :the-var
//	   :env  env
//	   :form form
//	   :var  var}
//	  (throw (ex-info (str "var not found: " var) {:var var}))))
func (a *Analyzer) parseVar(form interface{}, env Env) (*ast.Node2, error) {
	vrSym := second(form)
	if Count(form) != 2 {
		return nil, exInfo(fmt.Sprintf("wrong number of args to var, had: %d", Count(form)-1), nil)
	}
	maybeVar := a.resolveSym(vrSym, env)
	if maybeVar == nil {
		return nil, exInfo(fmt.Sprintf("var not found: %s", vrSym), nil)
	}
	vr, ok := maybeVar.(*lang.Var)
	if !ok {
		return nil, exInfo(fmt.Sprintf("expecting var, but %s is mapped to %v", vrSym, maybeVar), nil)
	}
	n := ast.MakeNode2(ast.OpTheVar, form)
	n.Env = env
	n.Sub = &ast.TheVarNode{
		Var: vr,
	}
	return n, nil
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
func (a *Analyzer) wrappingMeta(expr *ast.Node2) (*ast.Node2, error) {
	form := expr.Form
	env := expr.Env
	var meta IPersistentMap
	if m, ok := form.(IMeta); ok {
		meta = m.Meta()
	}
	if Seq(meta) == nil {
		return expr, nil
	}
	metaNode, err := a.analyzeForm(meta, ctxEnv(env, ctxExpr))
	if err != nil {
		return nil, err
	}
	expr.Env = lang.Assoc(expr.Env, KWContext, ctxExpr).(Env)
	n := ast.MakeNode2(ast.OpWithMeta, form)
	n.Env = env
	n.Sub = &ast.WithMetaNode{
		Expr: expr,
		Meta: metaNode,
	}
	return n, nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers

// (defn resolve-ns
//
//	"Resolves the ns mapped by the given sym in the global env"
//	[ns-sym {:keys [ns]}]
//	(when ns-sym
//	  (let [namespaces (:namespaces (env/deref-env))]
//	    (or (get-in namespaces [ns :aliases ns-sym])
//	        (:ns (namespaces ns-sym))))))
func (a *Analyzer) resolveNS(nsSym *lang.Symbol, env Env) *Symbol {
	curNSSym := Get(env, KWNS).(*lang.Symbol)
	if nsSym == nil {
		return nil
	}
	curNS := a.FindNamespace(curNSSym)
	if curNS == nil {
		return nil
	}
	if res := curNS.LookupAlias(nsSym); res != nil {
		return res.Name()
	}
	if ns := a.FindNamespace(nsSym); ns != nil {
		return ns.Name()
	}
	return nil
}

// (defn resolve-sym
//
//	"Resolves the value mapped by the given sym in the global env"
//	[sym {:keys [ns] :as env}]
//	(when (symbol? sym)
//	  (let [sym-ns (when-let [ns (namespace sym)]
//	                 (symbol ns))
//	        full-ns (resolve-ns sym-ns env)]
//	    (when (or (not sym-ns) full-ns)
//	      (let [name (if sym-ns (-> sym name symbol) sym)]
//	        (-> (env/deref-env) :namespaces (get (or full-ns ns)) :mappings (get name)))))))
func (a *Analyzer) resolveSym(symIfc interface{}, env Env) interface{} {
	ns, _ := Get(env, KWNS).(*Symbol)

	sym, ok := symIfc.(*Symbol)
	if !ok {
		return nil
	}
	var symNS *Symbol
	if sym.Namespace() != "" {
		symNS = NewSymbol(sym.Namespace())
	}
	fullNS := a.resolveNS(symNS, env)
	if fullNS == nil && symNS != nil {
		return nil
	}

	var name *Symbol
	if symNS != nil {
		name = NewSymbol(sym.Name())
	} else {
		name = sym
	}

	if fullNS != nil {
		ns = fullNS
	}
	theNS := a.FindNamespace(ns)
	return Get(theNS.Mappings(), name)
}

// (defn validate-bindings
//
//	[[op bindings & _ :as form] env]
//	(when-let [error-msg
//	           (cond
//	            (not (vector? bindings))
//	            (str op " requires a vector for its bindings, had: "
//	                 (class bindings))
//	            (not (even? (count bindings)))
//	            (str op " requires an even number of forms in binding vector, had: "
//	                 (count bindings)))]
//	  (throw (ex-info error-msg
//	                  (merge {:form     form
//	                          :bindings bindings}
//	                         (-source-info form env))))))
func (a *Analyzer) validateBindings(form interface{}, env Env) error {
	op := First(form)
	bindings, ok := second(form).(IPersistentVector)
	errMsg := ""
	switch {
	case !ok:
		errMsg = fmt.Sprintf("%s requires a vector for its bindings, had: %T", op, bindings)
	case Count(bindings)%2 != 0:
		errMsg = fmt.Sprintf("%s requires an even number of forms in binding vector, had: %d", op, Count(bindings))
	}
	if errMsg == "" {
		return nil
	}
	return exInfo(errMsg, nil)
}

func vec(v ...interface{}) *Vector {
	return NewVector(v...)
}

func second(x interface{}) interface{} {
	return First(Rest(x))
}

func exInfo(errStr string, _ interface{}) error {
	// TODO
	return fmt.Errorf(errStr)
}

func withRawForm(n *ast.Node2, form interface{}) *ast.Node2 {
	n.RawForms = append(n.RawForms, form)
	return n
}

func mapWhen(k, v interface{}) IPersistentMap {
	if v == nil {
		return nil
	}
	return NewMap(k, v)
}

func merge(as ...interface{}) Associative {
	if len(as) == 0 {
		return nil
	}

	var res Associative
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
	res = as[i].(Associative)
	for _, a := range as[i+1:] {
		for seq := Seq(a); seq != nil; seq = seq.Next() {
			entry := seq.First().(IMapEntry)
			res = Assoc(res, entry.Key(), entry.Val())
		}
	}
	return res
}

func remove(v interface{}, coll interface{}) interface{} {
	if coll == nil {
		return nil
	}
	var items []interface{}
	for seq := Seq(coll); seq != nil; seq = seq.Next() {
		if !Equal(v, seq.First()) {
			items = append(items, seq.First())
		}
	}
	return vec(items...)
}

func removeP(fn func(interface{}) bool, coll interface{}) interface{} {
	if coll == nil {
		return nil
	}
	var items []interface{}
	for seq := Seq(coll); seq != nil; seq = seq.Next() {
		if !fn(seq.First()) {
			items = append(items, seq.First())
		}
	}
	return vec(items...)
}

func ctxEnv(env Env, ctx Keyword) Env {
	return Assoc(env, KWContext, ctx).(Env)
}

func assocIn(mp interface{}, keys interface{}, val interface{}) Associative {
	count := Count(keys)
	if count == 0 {
		return Assoc(mp, nil, val)
	} else if count == 1 {
		return Assoc(mp, First(keys), val)
	}
	return Assoc(mp, First(keys),
		assocIn(Get(mp, First(keys)), Rest(keys), val))
}

func updateIn(mp interface{}, keys interface{}, fn func(...interface{}) Associative, args ...interface{}) Associative {
	var up func(mp interface{}, keys interface{}, fn func(...interface{}) Associative, args []interface{}) Associative
	up = func(mp interface{}, keys interface{}, fn func(...interface{}) Associative, args []interface{}) Associative {
		k := First(keys)
		ks := Rest(keys)
		if Count(ks) > 0 {
			return Assoc(mp, k, up(Get(mp, k), ks, fn, args))
		} else {
			vals := []interface{}{Get(mp, k)}
			vals = append(vals, args...)
			return Assoc(mp, k, fn(vals...))
		}
	}
	return up(mp, keys, fn, args)
}

func dissocEnv(node interface{}) interface{} {
	n := node.(*ast.Node2)
	newN := *n
	newN.Env = nil
	return &newN
}

func isValidBindingSymbol(v interface{}) bool {
	sym, ok := v.(*Symbol)
	if !ok {
		return false
	}
	if sym.Namespace() != "" {
		return false
	}
	if strings.Contains(sym.Name(), ".") {
		return false
	}
	return true
}

func classifyType(v interface{}) Keyword {
	switch v.(type) {
	case nil:
		return KWNil
	case bool:
		return KWBool
	case Keyword:
		return KWKeyword
	case *Symbol:
		return KWSymbol
	case string:
		return KWString
	case IPersistentVector:
		return KWVector
	case IPersistentMap:
		return KWMap
	case IPersistentSet:
		return KWSet
	case ISeq:
		return KWSeq
	case *Char:
		return KWChar
	case *regexp.Regexp:
		return KWRegex
	case *Var:
		return KWVar
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		*BigInt, *BigDecimal, *Ratio:
		return KWNumber
	default:
		return KWUnknown

		// TODO: type, record, class
	}
}

// (defn ^:private split-with' [pred coll]
//
//	(loop [take [] drop coll]
//	  (if (seq drop)
//	    (let [[el & r] drop]
//	      (if (pred el)
//	        (recur (conj take el) r)
//	        [(seq take) drop]))
//	    [(seq take) ()])))
func splitWith(pred func(interface{}) bool, coll interface{}) (interface{}, interface{}) {
	var take Conjer = vec()
	drop := coll
	for {
		seq := Seq(drop)
		if seq == nil {
			return Seq(take), NewList()
		}
		el := First(drop)
		if pred(el) {
			take = Conj(take, el)
			drop = Rest(drop)
		} else {
			return Seq(take), drop
		}
	}
}

func updateVals(m interface{}, fn func(interface{}) interface{}) interface{} {
	in := m
	if in == nil {
		in = NewMap()
	}
	return ReduceKV(func(m interface{}, k interface{}, v interface{}) interface{} {
		return Assoc(m, k, fn(v))
	}, NewMap(), in)
}

func unpackSeq(s interface{}, dsts ...interface{}) error {
	seq := Seq(s)
	for i, d := range dsts {
		if seq == nil {
			return fmt.Errorf("not enough arguments, got %d, expected %d", i, len(dsts))
		}
		dst := reflect.ValueOf(d)

		v := First(seq)
		if v == nil {
			if dst.Elem().Kind() != reflect.Interface {
				return fmt.Errorf("cannot assign nil to %s", dst.Type())
			}
			seq = seq.Next()
			continue // leave nil
		}

		val := reflect.ValueOf(v)
		seq = seq.Next()
		if !val.Type().AssignableTo(dst.Elem().Type()) {
			return fmt.Errorf("argument %d: expected %s, got %s", i, dst.Elem().Type(), val.Type())
		}
		dst.Elem().Set(val)
	}
	return nil
}
