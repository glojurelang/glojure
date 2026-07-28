package compiler

import (
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// IRValueKind is the representation-relevant type known for an AST value.
// Dynamic means that the analysis has not proved a narrower type.
type IRValueKind uint8

const (
	IRDynamic IRValueKind = iota
	IRNil
	IRBool
	IRInt
	IRFloat
	IRString
	IRKeyword
	IRVector
	IRMap
	IRSeq
	IRAtom
	IRFunction
)

// IREffect describes observable work performed while evaluating an IR node.
// Effects are deliberately conservative: an unknown call is both observable
// and potentially throwing.
type IREffect uint16

const (
	IREffectNone     IREffect = 0
	IREffectAllocate IREffect = 1 << iota
	IREffectReadVar
	IREffectReadMutable
	IREffectWriteMutable
	IREffectCallUnknown
	IREffectMayThrow
)

// IREscape describes whether a value allocated by a binding is observable
// outside the binding's synchronous lexical region.
type IREscape uint8

const (
	IREscapeUnknown IREscape = iota
	IRDoesNotEscape
	IREscapes
)

// IRShapeKind identifies collection layouts that can receive a concrete Go
// representation without changing Clojure's public value semantics.
type IRShapeKind uint8

const (
	IRShapeUnknown IRShapeKind = iota
	IRShapeVector
	IRShapeKeywordMap
)

type IRType struct {
	Kind     IRValueKind
	Nullable bool
	// GoType records a concrete Go representation when analysis can prove
	// one. Backends can use it at dynamic boundaries without recognizing the
	// source function or expression that produced the value.
	GoType reflect.Type
}

type IRShape struct {
	Kind     IRShapeKind
	Count    int
	Keywords []lang.Keyword
}

type IRCall struct {
	Var   *lang.Var
	Name  string
	Arity int
	Known bool
}

// IRGetInPlan represents (get-in value [k ...]) with a literal vector
// expression. The key expressions remain ordinary AST nodes so codegen
// preserves their left-to-right evaluation.
type IRGetInPlan struct {
	Target *ast.Node
	Keys   []*ast.Node
}

type IRSwapPlan struct {
	Target   *ast.Node
	Callback *ast.Node
}

type IRStackAppendPlan struct {
	Stack *lang.Symbol
	Value *ast.Node
}

type IRStringJoinPlan struct {
	Separator *ast.Node
	Head      *ast.Node
	Stack     *lang.Symbol
}

// IROwnedMapReducePlan represents a reduce whose inline callback carries one
// map accumulator through a chain of update-in calls without exposing any
// intermediate map identity.
type IROwnedMapReducePlan struct {
	ReduceVar    *lang.Var
	UpdateInVars []*lang.Var
	Reducer      *ast.Node
	Initial      *ast.Node
	Source       *ast.Node
	Updates      []*ast.Node
}

type IRPipelineConsumerKind uint8

const (
	IRPipelineSeq IRPipelineConsumerKind = iota
	IRPipelineReduce
	IRPipelineMapv
	IRPipelineInto
)

type IRPipelineStageKind uint8

const (
	IRPipelineMap IRPipelineStageKind = iota
	IRPipelineFilter
	IRPipelineTake
)

// IRPipelinePrimitive names callbacks whose exact Clojure semantics are
// understood by a lowering. Unknown callbacks remain represented in the
// pipeline but cannot be lowered without invoking the original function.
type IRPipelinePrimitive uint8

const (
	IRPipelinePrimitiveUnknown IRPipelinePrimitive = iota
	IRPipelineMapIdentity
	IRPipelineMapInc
	IRPipelineMapDec
	IRPipelineMapSquare
	IRPipelineFilterOdd
	IRPipelineFilterEven
	IRPipelineFilterPos
	IRPipelineFilterNeg
	IRPipelineFilterZero
)

type IRPipelineLowering uint8

const (
	IRPipelineNoLowering IRPipelineLowering = iota
	IRPipelineReduceInt64
)

type IRPipelineStage struct {
	Kind        IRPipelineStageKind
	Callback    *ast.Node
	Limit       *ast.Node
	OperatorVar *lang.Var
	CallbackVar *lang.Var
	Primitive   IRPipelinePrimitive
}

// IRPipelinePlan describes a collection pipeline independently of either the
// evaluator or AOT backend. A plan may intentionally have NoLowering: this
// lets optimizers reason about general pipelines without enabling a rewrite
// until benchmarks and semantic checks justify it.
type IRPipelinePlan struct {
	Consumer    IRPipelineConsumerKind
	ConsumerVar *lang.Var
	Reducer     *ast.Node
	Initial     *ast.Node
	IntoTarget  *ast.Node
	Source      *ast.Node
	Stages      []IRPipelineStage
	GuardVars   []*lang.Var
	Lowering    IRPipelineLowering
	TakeLimit   int64
}

type IRFacts struct {
	Type     IRType
	Effects  IREffect
	Escape   IREscape
	Shape    IRShape
	Call     IRCall
	GetIn    *IRGetInPlan
	Swap     *IRSwapPlan
	Append   *IRStackAppendPlan
	Join     *IRStringJoinPlan
	Pipeline *IRPipelinePlan

	OwnedMapReduce *IROwnedMapReducePlan

	// OwnedMapUpdateIn marks an update-in call inside an owned map reduction.
	// Backends may mutate the private accumulator representation while
	// preserving the persistent map observed by callbacks and at the boundary.
	OwnedMapUpdateIn bool

	// OwnedMapAssoc marks an assoc whose target is a uniquely owned
	// loop-carried map. Backends may update a transient representation after
	// evaluating every key and value.
	OwnedMapAssoc bool

	// OwnedMapGet marks a lookup against a uniquely owned loop-carried map.
	OwnedMapGet bool

	// PersistOwnedMap marks a terminal local whose transient representation
	// must be frozen before it leaves its ownership region.
	PersistOwnedMap bool
}

type IRBindingFacts struct {
	Escape      IREscape
	AtomInit    *ast.Node
	StringStack bool
	OwnedMap    bool
	// StableType is a representation-relevant type proved to hold at loop
	// entry and at every recur edge for this binding.
	StableType IRType
}

// TypedIR is a side-effect-free analysis layer built after the shared AST
// optimizer. It retains AST identity for source/evaluator compatibility while
// giving every backend a common home for representation and lowering facts.
type TypedIR struct {
	root     *ast.Node
	facts    map[*ast.Node]IRFacts
	bindings map[*ast.Node]IRBindingFacts
}

func BuildTypedIR(root *ast.Node) *TypedIR {
	ir := &TypedIR{
		root:     root,
		facts:    make(map[*ast.Node]IRFacts),
		bindings: make(map[*ast.Node]IRBindingFacts),
	}
	ir.visit(root)
	for {
		propagated := ir.propagateLocalTypes(root, nil)
		refined := ir.refineTypes(root)
		if !propagated && !refined {
			break
		}
	}
	ir.analyzeBindings(root)
	return ir
}

func (ir *TypedIR) Root() *ast.Node {
	if ir == nil {
		return nil
	}
	return ir.root
}

func (ir *TypedIR) Facts(node *ast.Node) IRFacts {
	if ir == nil || node == nil {
		return IRFacts{}
	}
	return ir.facts[node]
}

func (ir *TypedIR) BindingFacts(binding *ast.Node) IRBindingFacts {
	if ir == nil || binding == nil {
		return IRBindingFacts{}
	}
	return ir.bindings[binding]
}

func (ir *TypedIR) propagateLocalTypes(
	node *ast.Node,
	scope map[string]IRType,
) bool {
	if node == nil {
		return false
	}
	if node.Op == ast.OpLocal {
		name := node.Sub.(*ast.LocalNode).Name
		if name != nil {
			if typ, ok := scope[name.String()]; ok {
				facts := ir.facts[node]
				if facts.Type != typ {
					facts.Type = typ
					ir.facts[node] = facts
					return true
				}
			}
		}
		return false
	}

	changed := false
	switch node.Op {
	case ast.OpLet, ast.OpLoop:
		let := node.Sub.(*ast.LetNode)
		current := irCopyTypeScope(scope)
		for _, binding := range let.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			changed = ir.propagateLocalTypes(bindingNode.Init, current) ||
				changed
			current[bindingNode.Name.String()] =
				ir.facts[bindingNode.Init].Type
		}
		return ir.propagateLocalTypes(let.Body, current) || changed
	case ast.OpFn:
		fn := node.Sub.(*ast.FnNode)
		for _, methodNode := range fn.Methods {
			method := methodNode.Sub.(*ast.FnMethodNode)
			methodScope := irCopyTypeScope(scope)
			for _, param := range method.Params {
				name := param.Sub.(*ast.BindingNode).Name
				methodScope[name.String()] =
					IRType{Kind: IRDynamic, Nullable: true}
			}
			changed = ir.propagateLocalTypes(method.Body, methodScope) ||
				changed
		}
		return changed
	case ast.OpFnMethod:
		method := node.Sub.(*ast.FnMethodNode)
		methodScope := irCopyTypeScope(scope)
		for _, param := range method.Params {
			name := param.Sub.(*ast.BindingNode).Name
			methodScope[name.String()] =
				IRType{Kind: IRDynamic, Nullable: true}
		}
		return ir.propagateLocalTypes(method.Body, methodScope)
	}
	for _, child := range irChildren(node) {
		changed = ir.propagateLocalTypes(child, scope) || changed
	}
	return changed
}

// refineTypes recomputes facts whose result type depends on child facts. It is
// run to a fixed point with local propagation so nested lets and loops can
// acquire useful types without making the AST optimizer backend-specific.
func (ir *TypedIR) refineTypes(node *ast.Node) bool {
	if node == nil {
		return false
	}
	changed := false
	for _, child := range irChildren(node) {
		changed = ir.refineTypes(child) || changed
	}
	facts := ir.facts[node]
	typ := facts.Type
	switch node.Op {
	case ast.OpDo:
		typ = ir.facts[node.Sub.(*ast.DoNode).Ret].Type
	case ast.OpLet, ast.OpLoop:
		typ = ir.facts[node.Sub.(*ast.LetNode).Body].Type
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		thenType := ir.facts[ifNode.Then].Type
		elseType := ir.facts[ifNode.Else].Type
		if thenType.Kind == elseType.Kind {
			goType := thenType.GoType
			if goType != elseType.GoType {
				goType = nil
			}
			typ = IRType{
				Kind:     thenType.Kind,
				Nullable: thenType.Nullable || elseType.Nullable,
				GoType:   goType,
			}
		}
	case ast.OpAssoc:
		if ir.facts[node.Sub.(*ast.AssocNode).Target].Type.Kind == IRMap {
			typ = IRType{Kind: IRMap}
		}
	case ast.OpHostCall:
		if inferred, ok := ir.inferHostCallType(
			node.Sub.(*ast.HostCallNode),
		); ok {
			typ = inferred
		} else if inferred, ok := irResolvedCallType(
			node.Sub.(*ast.HostCallNode).ResolvedMethod,
		); ok {
			typ = inferred
		}
	}
	if facts.Type != typ {
		facts.Type = typ
		ir.facts[node] = facts
		changed = true
	}
	return changed
}

func (ir *TypedIR) inferHostCallType(
	call *ast.HostCallNode,
) (IRType, bool) {
	if call == nil || call.Method == nil ||
		call.Target == nil || call.Target.Op != ast.OpConst {
		return IRType{}, false
	}
	target := call.Target.Sub.(*ast.ConstNode)
	if target.Value != lang.Numbers {
		if target.HostSymbol == nil ||
			target.HostSymbol.String() !=
				"github.com:glojurelang:glojure:pkg:lang.Numbers" {
			return IRType{}, false
		}
	}
	allInt := len(call.Args) > 0
	for _, argument := range call.Args {
		if ir.facts[argument].Type.Kind != IRInt {
			allInt = false
			break
		}
	}
	if !allInt {
		return IRType{}, false
	}
	switch strings.ToLower(call.Method.Name()) {
	case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec",
		"add", "uncheckedadd", "minus", "unchecked_minus", "multiply",
		"unchecked_multiply", "quotient", "remainder":
		return IRType{Kind: IRInt}, true
	case "lt", "lte", "gt", "gte", "iszero", "ispos", "isneg":
		return IRType{Kind: IRBool}, true
	default:
		return IRType{}, false
	}
}

func irCopyTypeScope(scope map[string]IRType) map[string]IRType {
	result := make(map[string]IRType, len(scope)+1)
	for name, typ := range scope {
		result[name] = typ
	}
	return result
}

func (ir *TypedIR) visit(node *ast.Node) {
	if node == nil {
		return
	}
	children := irChildren(node)
	for _, child := range children {
		ir.visit(child)
	}

	facts := IRFacts{
		Type:   IRType{Kind: IRDynamic, Nullable: true},
		Escape: IREscapeUnknown,
	}
	for _, child := range children {
		facts.Effects |= ir.facts[child].Effects
	}

	switch node.Op {
	case ast.OpConst:
		facts.Type = irConstType(node.Sub.(*ast.ConstNode).Value)
	case ast.OpVector:
		vector := node.Sub.(*ast.VectorNode)
		facts.Type = IRType{Kind: IRVector}
		facts.Shape = IRShape{Kind: IRShapeVector, Count: len(vector.Items)}
		facts.Effects |= IREffectAllocate
	case ast.OpMap:
		m := node.Sub.(*ast.MapNode)
		facts.Type = IRType{Kind: IRMap}
		facts.Effects |= IREffectAllocate
		if keywords, ok := irKeywordKeys(m.Keys); ok {
			facts.Shape = IRShape{
				Kind:     IRShapeKeywordMap,
				Count:    len(keywords),
				Keywords: keywords,
			}
		}
	case ast.OpFn:
		facts.Type = IRType{Kind: IRFunction}
		facts.Effects |= IREffectAllocate
	case ast.OpDo:
		facts.Type = ir.facts[node.Sub.(*ast.DoNode).Ret].Type
	case ast.OpLet, ast.OpLoop:
		facts.Type = ir.facts[node.Sub.(*ast.LetNode).Body].Type
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		thenType := ir.facts[ifNode.Then].Type
		elseType := ir.facts[ifNode.Else].Type
		if thenType.Kind == elseType.Kind {
			goType := thenType.GoType
			if goType != elseType.GoType {
				goType = nil
			}
			facts.Type = IRType{
				Kind:     thenType.Kind,
				Nullable: thenType.Nullable || elseType.Nullable,
				GoType:   goType,
			}
		}
	case ast.OpHostCall:
		if inferred, ok := irResolvedCallType(
			node.Sub.(*ast.HostCallNode).ResolvedMethod,
		); ok {
			facts.Type = inferred
		}
	case ast.OpVar:
		facts.Effects |= IREffectReadVar
	case ast.OpKeywordLookup:
		facts.Effects |= IREffectMayThrow
	case ast.OpInvoke:
		ir.inferInvoke(node, &facts)
		facts.Pipeline = AnalyzePipeline(node)
	}
	ir.facts[node] = facts
	if node.Op == ast.OpInvoke {
		if plan := ir.analyzeOwnedMapReduce(node); plan != nil {
			facts = ir.facts[node]
			facts.OwnedMapReduce = plan
			facts.Type = IRType{Kind: IRMap}
			ir.facts[node] = facts
		}
	}
}

func (ir *TypedIR) inferInvoke(node *ast.Node, facts *IRFacts) {
	invoke := node.Sub.(*ast.InvokeNode)
	if invoke.Fn != nil && invoke.Fn.Op == ast.OpConst {
		if inferred, ok := irResolvedCallType(
			invoke.Fn.Sub.(*ast.ConstNode).Value,
		); ok {
			facts.Type = inferred
		}
	}
	vr, namespace, name, known := irVarCall(invoke)
	facts.Call = IRCall{
		Var:   vr,
		Name:  name,
		Arity: len(invoke.Args),
		Known: known,
	}
	if !known {
		facts.Effects |= IREffectCallUnknown | IREffectMayThrow
		return
	}

	if namespace == "clojure.string" {
		switch name {
		case "join", "replace":
			facts.Type = IRType{Kind: IRString}
		}
		facts.Effects |= IREffectMayThrow
		return
	}
	if namespace != "clojure.core" {
		facts.Effects |= IREffectCallUnknown | IREffectMayThrow
		return
	}

	switch name {
	case "atom":
		facts.Type = IRType{Kind: IRAtom}
		facts.Effects |= IREffectAllocate
	case "count":
		facts.Type = IRType{Kind: IRInt}
	case "str":
		facts.Type = IRType{Kind: IRString}
	case "subs":
		facts.Type = IRType{Kind: IRString}
		facts.Effects |= IREffectMayThrow
	case "=":
		facts.Type = IRType{Kind: IRBool}
	case "deref":
		facts.Effects |= IREffectReadMutable
	case "reset!", "swap!":
		facts.Effects |= IREffectWriteMutable | IREffectMayThrow
		if name == "swap!" && len(invoke.Args) == 2 &&
			irFixedFnArity(invoke.Args[1], 1) {
			facts.Swap = &IRSwapPlan{
				Target:   invoke.Args[0],
				Callback: invoke.Args[1],
			}
		}
	case "get-in":
		facts.Effects |= IREffectMayThrow
		if len(invoke.Args) == 2 && invoke.Args[1].Op == ast.OpVector {
			keys := invoke.Args[1].Sub.(*ast.VectorNode).Items
			facts.GetIn = &IRGetInPlan{
				Target: invoke.Args[0],
				Keys:   append([]*ast.Node(nil), keys...),
			}
		}
	default:
		// A recognized core call is still conservatively observable until a
		// lowering supplies a more precise summary.
		facts.Effects |= IREffectMayThrow
	}
}

func irFixedFnArity(node *ast.Node, arity int) bool {
	if node == nil || node.Op != ast.OpFn {
		return false
	}
	fn := node.Sub.(*ast.FnNode)
	if fn.IsVariadic || fn.Local != nil || len(fn.Methods) != 1 {
		return false
	}
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	return !method.IsVariadic && method.FixedArity == arity
}

func (ir *TypedIR) analyzeBindings(node *ast.Node) {
	if node == nil {
		return
	}
	if node.Op == ast.OpLet || node.Op == ast.OpLoop {
		let := node.Sub.(*ast.LetNode)
		for i, binding := range let.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			result := IRBindingFacts{
				Escape:     IREscapes,
				StableType: ir.facts[bindingNode.Init].Type,
			}
			if init := irScalarAtomInit(
				bindingNode,
				let.Bindings[i+1:],
				let.Body,
			); init != nil {
				result.Escape = IRDoesNotEscape
				result.AtomInit = init
			}
			if ir.analyzeStringStack(binding, let.Body) {
				result.Escape = IRDoesNotEscape
				result.StringStack = true
			}
			if node.Op == ast.OpLoop &&
				ir.analyzeOwnedMap(binding, i, let) {
				result.Escape = IRDoesNotEscape
				result.OwnedMap = true
			}
			if node.Op == ast.OpLoop {
				result.StableType = ir.analyzeStableLoopBinding(
					binding,
					i,
					let,
				)
			}
			ir.bindings[binding] = result
		}
	}
	for _, child := range irChildren(node) {
		ir.analyzeBindings(child)
	}
}

func (ir *TypedIR) analyzeStableLoopBinding(
	binding *ast.Node,
	bindingIndex int,
	loop *ast.LetNode,
) IRType {
	initType := ir.facts[binding.Sub.(*ast.BindingNode).Init].Type
	if initType.Kind == IRDynamic || initType.Kind == IRNil {
		return IRType{}
	}
	recurCount := 0
	stable := true
	var scan func(*ast.Node)
	scan = func(node *ast.Node) {
		if node == nil || !stable {
			return
		}
		// A nested function is a separate recurrence region.
		if node.Op == ast.OpFn || node.Op == ast.OpFnMethod {
			return
		}
		if node.Op == ast.OpRecur {
			recur := node.Sub.(*ast.RecurNode)
			if recur.LoopID == loop.LoopID {
				recurCount++
				if bindingIndex >= len(recur.Exprs) ||
					ir.facts[recur.Exprs[bindingIndex]].Type != initType {
					stable = false
				}
				return
			}
		}
		for _, child := range irChildren(node) {
			scan(child)
		}
	}
	scan(loop.Body)
	if !stable || recurCount == 0 {
		return IRType{}
	}
	return initType
}

func (ir *TypedIR) analyzeStringStack(
	binding *ast.Node,
	body *ast.Node,
) bool {
	bindingNode := binding.Sub.(*ast.BindingNode)
	if !ir.irEmptyVectorAtom(bindingNode.Init) {
		return false
	}

	allowed := make(map[*ast.Node]bool)
	discarded := irDiscardedNodes(body)
	type appendMatch struct {
		node  *ast.Node
		value *ast.Node
	}
	type joinMatch struct {
		node      *ast.Node
		separator *ast.Node
		head      *ast.Node
	}
	var appends []appendMatch
	var joins []joinMatch

	var scan func(*ast.Node)
	scan = func(node *ast.Node) {
		if node == nil {
			return
		}
		if target, value, ok := ir.matchStringStackAppend(
			node,
			bindingNode.Name,
		); ok {
			if !discarded[node] {
				return
			}
			allowed[target] = true
			appends = append(appends, appendMatch{node: node, value: value})
		}
		if local, separator, head, ok := ir.matchStringStackJoin(
			node,
			bindingNode.Name,
		); ok {
			allowed[local] = true
			joins = append(joins, joinMatch{
				node:      node,
				separator: separator,
				head:      head,
			})
		}
		for _, child := range irChildren(node) {
			scan(child)
		}
	}
	scan(body)
	if len(appends) == 0 || len(joins) == 0 {
		return false
	}

	safe := true
	var check func(*ast.Node)
	check = func(node *ast.Node) {
		if node == nil || !safe {
			return
		}
		if node.Op == ast.OpLocal &&
			node.Sub.(*ast.LocalNode).Name == bindingNode.Name &&
			!allowed[node] {
			safe = false
			return
		}
		for _, child := range irChildren(node) {
			check(child)
		}
	}
	check(body)
	if !safe {
		return false
	}

	for _, match := range appends {
		facts := ir.facts[match.node]
		facts.Append = &IRStackAppendPlan{
			Stack: bindingNode.Name,
			Value: match.value,
		}
		ir.facts[match.node] = facts
	}
	for _, match := range joins {
		facts := ir.facts[match.node]
		facts.Join = &IRStringJoinPlan{
			Separator: match.separator,
			Head:      match.head,
			Stack:     bindingNode.Name,
		}
		ir.facts[match.node] = facts
	}
	return true
}

func irDiscardedNodes(root *ast.Node) map[*ast.Node]bool {
	result := make(map[*ast.Node]bool)
	var walk func(*ast.Node, bool)
	walk = func(node *ast.Node, discarded bool) {
		if node == nil {
			return
		}
		if discarded {
			result[node] = true
		}
		switch node.Op {
		case ast.OpDo:
			do := node.Sub.(*ast.DoNode)
			for _, statement := range do.Statements {
				walk(statement, true)
			}
			walk(do.Ret, discarded)
		case ast.OpLet, ast.OpLoop:
			let := node.Sub.(*ast.LetNode)
			for _, binding := range let.Bindings {
				walk(binding.Sub.(*ast.BindingNode).Init, false)
			}
			walk(let.Body, discarded)
		case ast.OpIf:
			ifNode := node.Sub.(*ast.IfNode)
			walk(ifNode.Test, false)
			walk(ifNode.Then, discarded)
			walk(ifNode.Else, discarded)
		default:
			for _, child := range irChildren(node) {
				walk(child, false)
			}
		}
	}
	walk(root, false)
	return result
}

func (ir *TypedIR) irEmptyVectorAtom(node *ast.Node) bool {
	if node == nil || node.Op != ast.OpInvoke {
		return false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	return core && name == "atom" && len(invoke.Args) == 1 &&
		invoke.Args[0].Op == ast.OpVector &&
		len(invoke.Args[0].Sub.(*ast.VectorNode).Items) == 0
}

func (ir *TypedIR) matchStringStackAppend(
	node *ast.Node,
	stack *lang.Symbol,
) (*ast.Node, *ast.Node, bool) {
	if node == nil || node.Op != ast.OpInvoke {
		return nil, nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "swap!" || len(invoke.Args) != 2 ||
		!irLocalIs(invoke.Args[0], stack) ||
		!irFixedFnArity(invoke.Args[1], 1) {
		return nil, nil, false
	}
	fn := invoke.Args[1].Sub.(*ast.FnNode)
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	param := method.Params[0].Sub.(*ast.BindingNode).Name
	expression := irUnwrapDo(method.Body)
	if expression == nil || expression.Op != ast.OpInvoke {
		return nil, nil, false
	}
	cons := expression.Sub.(*ast.InvokeNode)
	_, consName, consCore := irCoreCall(cons)
	if !consCore || consName != "cons" || len(cons.Args) != 2 ||
		!irLocalIs(cons.Args[1], param) ||
		ir.facts[cons.Args[0]].Type.Kind != IRString {
		return nil, nil, false
	}
	return invoke.Args[0], cons.Args[0], true
}

func (ir *TypedIR) matchStringStackJoin(
	node *ast.Node,
	stack *lang.Symbol,
) (*ast.Node, *ast.Node, *ast.Node, bool) {
	if node == nil || node.Op != ast.OpInvoke {
		return nil, nil, nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	_, namespace, name, known := irVarCall(invoke)
	if !known || namespace != "clojure.string" || name != "join" ||
		len(invoke.Args) != 2 ||
		ir.facts[invoke.Args[0]].Type.Kind != IRString ||
		invoke.Args[1].Op != ast.OpInvoke {
		return nil, nil, nil, false
	}
	cons := invoke.Args[1].Sub.(*ast.InvokeNode)
	_, consName, consCore := irCoreCall(cons)
	if !consCore || consName != "cons" || len(cons.Args) != 2 ||
		cons.Args[1].Op != ast.OpInvoke {
		return nil, nil, nil, false
	}
	deref := cons.Args[1].Sub.(*ast.InvokeNode)
	_, derefName, derefCore := irCoreCall(deref)
	if !derefCore || derefName != "deref" || len(deref.Args) != 1 ||
		!irLocalIs(deref.Args[0], stack) {
		return nil, nil, nil, false
	}
	return deref.Args[0], invoke.Args[0], cons.Args[0], true
}

func irUnwrapDo(node *ast.Node) *ast.Node {
	for node != nil && node.Op == ast.OpDo {
		do := node.Sub.(*ast.DoNode)
		if len(do.Statements) != 0 {
			return nil
		}
		node = do.Ret
	}
	return node
}

func irLocalIs(node *ast.Node, name *lang.Symbol) bool {
	return node != nil && node.Op == ast.OpLocal &&
		node.Sub.(*ast.LocalNode).Name == name
}

func irConstType(value any) IRType {
	switch value.(type) {
	case nil:
		return IRType{Kind: IRNil, Nullable: true}
	case bool:
		return IRType{Kind: IRBool}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return IRType{Kind: IRInt}
	case float32, float64:
		return IRType{Kind: IRFloat}
	case string:
		return IRType{Kind: IRString}
	case lang.Keyword:
		return IRType{Kind: IRKeyword}
	default:
		return IRType{Kind: IRDynamic, Nullable: true}
	}
}

func irResolvedCallType(callable any) (IRType, bool) {
	typ := reflect.TypeOf(callable)
	if typ == nil || typ.Kind() != reflect.Func || typ.NumOut() != 1 {
		return IRType{}, false
	}
	result := typ.Out(0)
	inferred := IRType{GoType: result}
	switch result.Kind() {
	case reflect.Bool:
		inferred.Kind = IRBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		inferred.Kind = IRInt
	case reflect.Float32, reflect.Float64:
		inferred.Kind = IRFloat
	case reflect.String:
		inferred.Kind = IRString
	default:
		return IRType{}, false
	}
	return inferred, true
}

func irKeywordKeys(keys []*ast.Node) ([]lang.Keyword, bool) {
	result := make([]lang.Keyword, len(keys))
	for i, key := range keys {
		if key == nil || key.Op != ast.OpConst {
			return nil, false
		}
		keyword, ok := key.Sub.(*ast.ConstNode).Value.(lang.Keyword)
		if !ok {
			return nil, false
		}
		result[i] = keyword
	}
	return result, true
}

func irVarCall(invoke *ast.InvokeNode) (*lang.Var, string, string, bool) {
	if invoke == nil || invoke.Fn == nil || invoke.Fn.Op != ast.OpVar {
		return nil, "", "", false
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr == nil || vr.Namespace() == nil {
		return nil, "", "", false
	}
	return vr, vr.Namespace().Name().String(), vr.Symbol().String(), true
}

func irCoreCall(invoke *ast.InvokeNode) (*lang.Var, string, bool) {
	vr, namespace, name, ok := irVarCall(invoke)
	return vr, name, ok && namespace == "clojure.core"
}

var (
	irASTNodePointerType = reflect.TypeOf((*ast.Node)(nil))
	irASTPackagePath     = reflect.TypeOf(ast.Node{}).PkgPath()
)

func irChildren(node *ast.Node) []*ast.Node {
	if node == nil {
		return nil
	}
	var result []*ast.Node
	var walk func(reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Type() == irASTNodePointerType {
			if !value.IsNil() {
				result = append(result, value.Interface().(*ast.Node))
			}
			return
		}
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if !value.IsNil() {
				element := value.Elem()
				if element.Kind() == reflect.Struct &&
					element.Type().PkgPath() == irASTPackagePath {
					walk(element)
				}
			}
		case reflect.Struct:
			if value.Type().PkgPath() != irASTPackagePath {
				return
			}
			for i := 0; i < value.NumField(); i++ {
				walk(value.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node.Sub))
	return result
}

func irScalarAtomInit(
	binding *ast.BindingNode,
	laterBindings []*ast.Node,
	body *ast.Node,
) *ast.Node {
	init := binding.Init
	if init == nil || init.Op != ast.OpInvoke {
		return nil
	}
	invoke := init.Sub.(*ast.InvokeNode)
	_, name, core := irCoreCall(invoke)
	if !core || name != "atom" || len(invoke.Args) != 1 {
		return nil
	}
	usage := irLocalAtomUsage{target: binding.Name, safe: true}
	for _, later := range laterBindings {
		usage.walk(later.Sub.(*ast.BindingNode).Init, false)
	}
	usage.walk(body, false)
	if !usage.safe || usage.uses == 0 {
		return nil
	}
	return invoke.Args[0]
}

type irLocalAtomUsage struct {
	target *lang.Symbol
	safe   bool
	uses   int
}

func (u *irLocalAtomUsage) walk(node *ast.Node, inFunction bool) {
	if node == nil || !u.safe {
		return
	}
	if u.isTarget(node) {
		u.uses++
		u.safe = false
		return
	}
	if node.Op == ast.OpFn || node.Op == ast.OpFnMethod || node.Op == ast.OpGo {
		for _, child := range irChildren(node) {
			u.walk(child, true)
		}
		return
	}
	if node.Op == ast.OpInvoke {
		invoke := node.Sub.(*ast.InvokeNode)
		_, operation, core := irCoreCall(invoke)
		if core && (operation == "deref" ||
			operation == "reset!" || operation == "swap!") &&
			len(invoke.Args) > 0 && u.isTarget(invoke.Args[0]) {
			u.uses++
			if inFunction {
				u.safe = false
				return
			}
			u.walk(invoke.Fn, inFunction)
			for _, arg := range invoke.Args[1:] {
				u.walk(arg, inFunction)
			}
			return
		}
	}
	for _, child := range irChildren(node) {
		u.walk(child, inFunction)
	}
}

func (u *irLocalAtomUsage) isTarget(node *ast.Node) bool {
	return node != nil && node.Op == ast.OpLocal &&
		node.Sub.(*ast.LocalNode).Name == u.target
}
