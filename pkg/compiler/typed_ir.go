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

// Accepts reports whether actual satisfies an expected representation fact.
// A fully dynamic expectation is a wildcard. An expected concrete Go type is
// exact and must have been established explicitly on the actual value.
func (expected IRType) Accepts(actual IRType) bool {
	if expected.Kind == IRDynamic && expected.GoType == nil {
		return true
	}
	if actual.Nullable && !expected.Nullable && expected.Kind != IRNil {
		return false
	}
	if expected.GoType != nil {
		// A semantic numeric kind is not a concrete Go representation. For
		// example, count returns int while arithmetic normally returns int64.
		// Exact consumers therefore require an explicitly proved GoType rather
		// than filling in a convenient canonical representation.
		if expected.Kind != IRDynamic && expected.Kind != actual.Kind {
			return false
		}
		return expected.GoType == actual.GoType
	}
	return expected.Kind == actual.Kind
}

func canonicalIRGoType(kind IRValueKind) reflect.Type {
	switch kind {
	case IRBool:
		return reflect.TypeOf(false)
	case IRInt:
		return reflect.TypeOf(int64(0))
	case IRFloat:
		return reflect.TypeOf(float64(0))
	case IRString:
		return reflect.TypeOf("")
	case IRKeyword:
		return reflect.TypeOf(lang.Keyword{})
	default:
		return nil
	}
}

type IRShape struct {
	Kind     IRShapeKind
	Count    int
	Keywords []lang.Keyword
}

// IROwnedMapMode describes the internal representation selected for a
// non-escaping loop-carried map.
type IROwnedMapMode uint8

const (
	IROwnedMapNone IROwnedMapMode = iota
	// IROwnedMapTransient is used when the loop initializer is already known
	// to be a persistent map.
	IROwnedMapTransient
	// IROwnedMapAdaptive keeps a dynamically typed initializer on the
	// persistent path until its first effective map update.
	IROwnedMapAdaptive
)

type IRCall struct {
	Var   *lang.Var
	Name  string
	Arity int
	Known bool
}

type IRCallSignature struct {
	Params []IRType
	Result IRType
}

// IRDirectCallSite exposes the argument representations inferred at a
// statically resolved Var call. Backends can use these shared facts to prepare
// typed entry points without rediscovering caller-local types.
type IRDirectCallSite struct {
	Node          *ast.Node
	Var           *lang.Var
	ArgumentTypes []IRType
}

type IRFixedDispatchResultKind uint8

const (
	IRDispatchScalar IRFixedDispatchResultKind = iota
	IRDispatchVector
)

// IRFixedDispatchResultPlan describes a fixed-arity function whose body
// returns either a scalar expression or a vector assembled directly from
// expressions over its parameters. A backend may consume vector components
// without materializing the vector when its identity cannot be observed.
type IRFixedDispatchResultPlan struct {
	Method     *ast.FnMethodNode
	Components []*ast.Node
	Kind       IRFixedDispatchResultKind
}

type TypedIROptions struct {
	CallSignatures map[*lang.Var][]IRCallSignature
	// ParameterTypes seeds representation facts for a guarded function
	// version. The ordinary unguarded analysis leaves every parameter
	// dynamic.
	ParameterTypes map[*lang.Symbol]IRType
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

// IROwnedStringPartsAppendPlan records a conj into a loop-carried vector
// whose only observable consumer is (apply str ...). The backing vector can
// be replaced by an append-only private parts buffer.
type IROwnedStringPartsAppendPlan struct {
	Parts *lang.Symbol
	Value *ast.Node
}

type IROwnedStringPartsFinishPlan struct {
	Parts *lang.Symbol
}

// IROwnedMapReducePlan represents a reduce whose inline callback linearly
// carries one map accumulator through ownership-compatible mutations without
// exposing any intermediate map identity.
type IROwnedMapReducePlan struct {
	ReduceVar *lang.Var
	CallVars  []*lang.Var
	Reducer   *ast.Node
	Initial   *ast.Node
	Source    *ast.Node
	Mutations []*ast.Node
}

type IROwnedMapMutationKind uint8

const (
	IROwnedMapUpdateIn IROwnedMapMutationKind = iota
	IROwnedMapAssoc
)

// IROwnedMapMutationPlan records an ownership-compatible map operation inside
// a reduction. Paths and assoc entries remain ordinary expressions; a backend
// can choose a representation-specific helper without changing eligibility.
type IROwnedMapMutationPlan struct {
	Kind IROwnedMapMutationKind
	Keys []*ast.Node
	Fnil *IROwnedMapFnilPlan
}

// IROwnedMapFnilPlan describes a one-default fnil wrapper whose identity is
// consumed only by the enclosing update-in. A backend may apply the default
// at the owned leaf when the recorded Var is safe to direct-link.
type IROwnedMapFnilPlan struct {
	Var     *lang.Var
	Fn      *ast.Node
	Default *ast.Node
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

type IRPipelineLowering uint8

const (
	IRPipelineNoLowering IRPipelineLowering = iota
	// IRPipelineInlineIndexed evaluates a literal callback directly while
	// traversing an Indexed source. Other source types retain the ordinary
	// collection-function fallback.
	IRPipelineInlineIndexed
)

type IRPipelineStage struct {
	Kind        IRPipelineStageKind
	Callback    *ast.Node
	Limit       *ast.Node
	OperatorVar *lang.Var
	CallbackVar *lang.Var
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
}

type IRFacts struct {
	Type    IRType
	Effects IREffect
	// NeverReturns means evaluating this expression cannot produce a value
	// for its enclosing expression. Throw and recur are the primitive cases;
	// compound control flow propagates the fact conservatively.
	NeverReturns      bool
	Escape            IREscape
	Shape             IRShape
	Call              IRCall
	Signature         *IRCallSignature
	GetIn             *IRGetInPlan
	Swap              *IRSwapPlan
	Append            *IRStackAppendPlan
	Join              *IRStringJoinPlan
	StringPartsAppend *IROwnedStringPartsAppendPlan
	StringPartsFinish *IROwnedStringPartsFinishPlan
	Pipeline          *IRPipelinePlan

	OwnedMapReduce *IROwnedMapReducePlan

	// OwnedMapMutation describes an operation inside an owned map reduction.
	// Backends may mutate the private accumulator representation while
	// preserving the persistent map observed by callbacks and at the boundary.
	OwnedMapMutation *IROwnedMapMutationPlan

	// OwnedMapAssoc marks an assoc whose target is a uniquely owned
	// loop-carried map. Backends may update a transient representation after
	// evaluating every key and value.
	OwnedMapAssoc bool

	// OwnedMapGet marks a lookup against a uniquely owned loop-carried map.
	OwnedMapGet bool

	// PersistOwnedMap marks a terminal local whose transient representation
	// must be frozen before it leaves its ownership region.
	PersistOwnedMap bool

	// OwnedMapMode identifies the representation expected by an owned-map
	// operation or terminal result.
	OwnedMapMode IROwnedMapMode
}

type IRBindingFacts struct {
	Escape           IREscape
	AtomInit         *ast.Node
	StringStack      bool
	OwnedStringParts bool
	OwnedMap         bool
	OwnedMapMode     IROwnedMapMode
	// StableType is a representation-relevant type proved to hold at loop
	// entry and at every recur edge for this binding.
	StableType IRType
}

// TypedIR is a side-effect-free analysis layer built after the shared AST
// optimizer. It retains AST identity for source/evaluator compatibility while
// giving every backend a common home for representation and lowering facts.
type TypedIR struct {
	root           *ast.Node
	facts          map[*ast.Node]IRFacts
	bindings       map[*ast.Node]IRBindingFacts
	callSignatures map[*lang.Var][]IRCallSignature
	parameterTypes map[*lang.Symbol]IRType
}

func BuildTypedIR(root *ast.Node) *TypedIR {
	return BuildTypedIRWithOptions(root, TypedIROptions{})
}

func BuildTypedIRWithOptions(
	root *ast.Node,
	options TypedIROptions,
) *TypedIR {
	ir := &TypedIR{
		root:           root,
		facts:          make(map[*ast.Node]IRFacts),
		bindings:       make(map[*ast.Node]IRBindingFacts),
		callSignatures: options.CallSignatures,
		parameterTypes: options.ParameterTypes,
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

func (ir *TypedIR) ResolvedCallVars() []*lang.Var {
	if ir == nil || ir.root == nil {
		return nil
	}
	vars := make(map[*lang.Var]struct{})
	var collect func(*ast.Node, bool)
	collect = func(node *ast.Node, root bool) {
		if node == nil {
			return
		}
		if node.Op == ast.OpFn {
			if !root {
				return
			}
			for _, methodNode := range node.Sub.(*ast.FnNode).Methods {
				method := methodNode.Sub.(*ast.FnMethodNode)
				collect(method.Body, false)
			}
			return
		}
		facts := ir.facts[node]
		if facts.Signature != nil && facts.Call.Var != nil {
			vars[facts.Call.Var] = struct{}{}
		}
		for _, child := range irChildren(node) {
			collect(child, false)
		}
	}
	collect(ir.root, true)
	if len(vars) == 0 {
		return nil
	}
	result := make([]*lang.Var, 0, len(vars))
	for vr := range vars {
		result = append(result, vr)
	}
	return result
}

func (ir *TypedIR) ResolvedCallCount() int {
	if ir == nil {
		return 0
	}
	count := 0
	for _, facts := range ir.facts {
		if facts.Signature != nil {
			count++
		}
	}
	return count
}

// RepresentationScore counts operations whose result has a concrete
// representation. Constants and local reads are excluded: the score is used
// to decide whether seeding parameter types unlocks useful work inside a
// function rather than merely restating the parameter representation.
func (ir *TypedIR) RepresentationScore() int {
	if ir == nil {
		return 0
	}
	score := 0
	for node, facts := range ir.facts {
		if facts.Type.Kind == IRDynamic {
			continue
		}
		switch node.Op {
		case ast.OpInvoke, ast.OpKeywordLookup, ast.OpAssoc,
			ast.OpReplaceLast, ast.OpHostCall, ast.OpHostField,
			ast.OpHostInterop, ast.OpIf, ast.OpCase:
			score++
		}
	}
	return score
}

// DirectCallSites returns statically resolved Var calls together with the
// representation inferred for each argument.
func (ir *TypedIR) DirectCallSites() []IRDirectCallSite {
	if ir == nil {
		return nil
	}
	var result []IRDirectCallSite
	for node, facts := range ir.facts {
		if node.Op != ast.OpInvoke || !facts.Call.Known ||
			facts.Call.Var == nil {
			continue
		}
		invoke := node.Sub.(*ast.InvokeNode)
		types := make([]IRType, len(invoke.Args))
		for index, argument := range invoke.Args {
			types[index] = ir.facts[argument].Type
		}
		result = append(result, IRDirectCallSite{
			Node:          node,
			Var:           facts.Call.Var,
			ArgumentTypes: types,
		})
	}
	return result
}

// AnalyzeFixedDispatchResult recognizes a composable region: one fixed-arity
// method returning a scalar expression or literal vector whose component
// expressions have no free locals or nested binding forms.
func AnalyzeFixedDispatchResult(root *ast.Node) *IRFixedDispatchResultPlan {
	if root == nil || root.Op != ast.OpFn {
		return nil
	}
	fn := root.Sub.(*ast.FnNode)
	if fn.IsVariadic || len(fn.Methods) != 1 || fn.Local != nil {
		return nil
	}
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	if method.IsVariadic || method.FixedArity < 0 ||
		method.FixedArity > 20 {
		return nil
	}
	body := method.Body
	if body.Op == ast.OpDo {
		do := body.Sub.(*ast.DoNode)
		if len(do.Statements) != 0 {
			return nil
		}
		body = do.Ret
	}
	if body == nil {
		return nil
	}
	kind := IRDispatchScalar
	components := []*ast.Node{body}
	if body.Op == ast.OpVector {
		kind = IRDispatchVector
		components = body.Sub.(*ast.VectorNode).Items
		if len(components) == 0 {
			return nil
		}
	}
	params := make(map[*lang.Symbol]struct{}, len(method.Params))
	for _, parameter := range method.Params {
		params[parameter.Sub.(*ast.BindingNode).Name] = struct{}{}
	}
	valid := true
	for _, component := range components {
		_, err := ast.Transform(component, func(node *ast.Node) (*ast.Node, error) {
			switch node.Op {
			case ast.OpLocal:
				if _, ok := params[node.Sub.(*ast.LocalNode).Name]; !ok {
					valid = false
				}
			case ast.OpFn, ast.OpFnMethod, ast.OpLet, ast.OpLetFn,
				ast.OpLoop, ast.OpRecur:
				valid = false
			}
			return node, nil
		})
		if err != nil || !valid {
			return nil
		}
	}
	return &IRFixedDispatchResultPlan{
		Method:     method,
		Components: append([]*ast.Node(nil), components...),
		Kind:       kind,
	}
}

// AnalyzeFixedVectorResult is retained for callers specifically requiring a
// vector result.
func AnalyzeFixedVectorResult(root *ast.Node) *IRFixedDispatchResultPlan {
	plan := AnalyzeFixedDispatchResult(root)
	if plan == nil || plan.Kind != IRDispatchVector {
		return nil
	}
	return plan
}

// GuardedCallsSafe reports whether every resolved call can use targets
// selected by a guard at function entry. Potentially mutating work only
// invalidates a resolved call when it can run before that call. Work after
// the final resolved call is irrelevant. Loops remain deliberately
// conservative because work later in one iteration can precede a resolved
// call in the next iteration.
func (ir *TypedIR) GuardedCallsSafe() bool {
	if ir == nil || ir.root == nil {
		return false
	}
	if ir.root.Op == ast.OpFn {
		for _, methodNode := range ir.root.Sub.(*ast.FnNode).Methods {
			method := methodNode.Sub.(*ast.FnMethodNode)
			if _, safe := ir.guardedCallsSafeBefore(
				method.Body,
				false,
			); !safe {
				return false
			}
		}
		return true
	}
	_, safe := ir.guardedCallsSafeBefore(ir.root, false)
	return safe
}

// guardedCallsSafeBefore walks evaluation order backwards. guardedAfter means
// that a resolved call can execute later on the current path. The returned
// boolean records whether a resolved call can execute before node.
func (ir *TypedIR) guardedCallsSafeBefore(
	node *ast.Node,
	guardedAfter bool,
) (guardedBefore bool, safe bool) {
	if node == nil {
		return guardedAfter, true
	}
	switch node.Op {
	case ast.OpFn:
		// Constructing a nested function does not execute its body. It will
		// receive its own guarded specialization when generated.
		return guardedAfter, true
	case ast.OpFnMethod:
		return ir.guardedCallsSafeBefore(
			node.Sub.(*ast.FnMethodNode).Body,
			guardedAfter,
		)
	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		need, ok := ir.guardedCallsSafeBefore(let.Body, guardedAfter)
		if !ok {
			return false, false
		}
		for index := len(let.Bindings) - 1; index >= 0; index-- {
			init := let.Bindings[index].Sub.(*ast.BindingNode).Init
			need, ok = ir.guardedCallsSafeBefore(init, need)
			if !ok {
				return false, false
			}
		}
		return need, true
	case ast.OpLoop:
		return ir.guardedLoopCallsSafeBefore(
			node.Sub.(*ast.LetNode),
			guardedAfter,
		)
	case ast.OpDo:
		do := node.Sub.(*ast.DoNode)
		need, ok := ir.guardedCallsSafeBefore(do.Ret, guardedAfter)
		if !ok {
			return false, false
		}
		for index := len(do.Statements) - 1; index >= 0; index-- {
			need, ok = ir.guardedCallsSafeBefore(
				do.Statements[index],
				need,
			)
			if !ok {
				return false, false
			}
		}
		return need, true
	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		thenNeed, ok := ir.guardedCallsSafeBefore(
			conditional.Then,
			guardedAfter,
		)
		if !ok {
			return false, false
		}
		elseNeed, ok := ir.guardedCallsSafeBefore(
			conditional.Else,
			guardedAfter,
		)
		if !ok {
			return false, false
		}
		return ir.guardedCallsSafeBefore(
			conditional.Test,
			thenNeed || elseNeed,
		)
	case ast.OpCase:
		caseNode := node.Sub.(*ast.CaseNode)
		branchNeed, ok := ir.guardedCallsSafeBefore(
			caseNode.Default,
			guardedAfter,
		)
		if !ok {
			return false, false
		}
		for _, entry := range caseNode.Entries {
			resultNeed, resultOK := ir.guardedCallsSafeBefore(
				entry.ResultExpr,
				guardedAfter,
			)
			if !resultOK {
				return false, false
			}
			branchNeed = branchNeed || resultNeed
		}
		return ir.guardedCallsSafeBefore(caseNode.Test, branchNeed)
	case ast.OpTry:
		tryNode := node.Sub.(*ast.TryNode)
		need, ok := ir.guardedCallsSafeBefore(
			tryNode.Finally,
			guardedAfter,
		)
		if !ok {
			return false, false
		}
		for _, catchNode := range tryNode.Catches {
			catch := catchNode.Sub.(*ast.CatchNode)
			catchNeed, catchOK := ir.guardedCallsSafeBefore(
				catch.Body,
				need,
			)
			if !catchOK {
				return false, false
			}
			need = need || catchNeed
		}
		return ir.guardedCallsSafeBefore(tryNode.Body, need)
	}

	need := guardedAfter
	facts := ir.facts[node]
	if facts.Signature != nil {
		need = true
	} else if ir.guardedCallUnsafe(node, facts) && need {
		return false, false
	}
	children := irGuardedEvaluationChildren(node)
	for index := len(children) - 1; index >= 0; index-- {
		var ok bool
		need, ok = ir.guardedCallsSafeBefore(children[index], need)
		if !ok {
			return false, false
		}
	}
	return need, true
}

func (ir *TypedIR) guardedLoopCallsSafeBefore(
	loop *ast.LetNode,
	guardedAfter bool,
) (bool, bool) {
	bodyGuarded, bodyUnsafe := ir.guardedLoopContents(loop.Body)
	if bodyGuarded && bodyUnsafe || bodyUnsafe && guardedAfter {
		return false, false
	}
	need := guardedAfter || bodyGuarded
	for index := len(loop.Bindings) - 1; index >= 0; index-- {
		init := loop.Bindings[index].Sub.(*ast.BindingNode).Init
		var ok bool
		need, ok = ir.guardedCallsSafeBefore(init, need)
		if !ok {
			return false, false
		}
	}
	return need, true
}

func (ir *TypedIR) guardedLoopContents(node *ast.Node) (
	guarded bool,
	unsafe bool,
) {
	if node == nil || node.Op == ast.OpFn {
		return false, false
	}
	facts := ir.facts[node]
	guarded = facts.Signature != nil
	unsafe = facts.Signature == nil && ir.guardedCallUnsafe(node, facts)
	for _, child := range irChildren(node) {
		childGuarded, childUnsafe := ir.guardedLoopContents(child)
		guarded = guarded || childGuarded
		unsafe = unsafe || childUnsafe
	}
	return guarded, unsafe
}

func (ir *TypedIR) guardedCallUnsafe(node *ast.Node, facts IRFacts) bool {
	switch node.Op {
	case ast.OpDef, ast.OpSetBang, ast.OpGo, ast.OpNew, ast.OpHostInterop:
		return true
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if call.Target == nil || call.Target.Op != ast.OpConst {
			return true
		}
		target := call.Target.Sub.(*ast.ConstNode)
		return target.Value != lang.Numbers &&
			(target.HostSymbol == nil ||
				target.HostSymbol.String() !=
					"github.com:glojurelang:glojure:pkg:lang.Numbers")
	case ast.OpInvoke:
		return facts.Signature == nil &&
			!irGuardedCoreCallSafe(facts.Call)
	default:
		return false
	}
}

func irGuardedEvaluationChildren(node *ast.Node) []*ast.Node {
	if node == nil {
		return nil
	}
	switch node.Op {
	case ast.OpMap:
		m := node.Sub.(*ast.MapNode)
		result := make([]*ast.Node, 0, len(m.Keys)+len(m.Vals))
		for index := range m.Keys {
			result = append(result, m.Keys[index], m.Vals[index])
		}
		return result
	case ast.OpAssoc:
		assoc := node.Sub.(*ast.AssocNode)
		result := make([]*ast.Node, 0, 1+2*len(assoc.Entries))
		result = append(result, assoc.Target)
		for _, entry := range assoc.Entries {
			result = append(result, entry.Key, entry.Val)
		}
		return result
	default:
		return irChildren(node)
	}
}

// irGuardedCoreCallSafe identifies calls which cannot replace a guarded
// target while the surrounding function is running. Keep this list
// intentionally narrow: being a known clojure.core Var does not imply that a
// call is read-only (extend and alter-var-root are counterexamples).
func irGuardedCoreCallSafe(call IRCall) bool {
	if !call.Known || call.Var == nil || call.Var.Namespace() == nil ||
		call.Var.Namespace().Name().String() != "clojure.core" {
		return false
	}
	switch call.Name {
	case "=", "==", "not=", "<", "<=", ">", ">=",
		"+", "-", "*", "/", "inc", "dec", "mod", "quot", "rem",
		"long", "int", "double", "boolean",
		"count", "get", "nth", "first", "next", "rest", "seq",
		"empty?", "nil?", "some?", "identical?":
		return true
	default:
		return false
	}
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
				methodScope[name.String()] = ir.parameterType(name)
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
			methodScope[name.String()] = ir.parameterType(name)
		}
		return ir.propagateLocalTypes(method.Body, methodScope)
	}
	for _, child := range irChildren(node) {
		changed = ir.propagateLocalTypes(child, scope) || changed
	}
	return changed
}

func (ir *TypedIR) parameterType(name *lang.Symbol) IRType {
	if ir != nil && name != nil {
		if typ, ok := ir.parameterTypes[name]; ok {
			return typ
		}
	}
	return IRType{Kind: IRDynamic, Nullable: true}
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
		typ = irJoinBranchTypes(
			ir.facts[ifNode.Then],
			ir.facts[ifNode.Else],
		)
	case ast.OpCase:
		typ = ir.caseResultType(node.Sub.(*ast.CaseNode))
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
	case ast.OpInvoke:
		if inferred, ok := ir.refineInvokeType(
			node.Sub.(*ast.InvokeNode),
			facts.Call,
		); ok {
			typ = inferred
		}
		if signature := ir.resolveCallSignature(
			node.Sub.(*ast.InvokeNode),
		); signature != nil {
			typ = signature.Result
			facts.Signature = signature
		} else {
			facts.Signature = nil
		}
	}
	if facts.Type != typ || ir.facts[node].Signature != facts.Signature {
		facts.Type = typ
		ir.facts[node] = facts
		changed = true
	}
	return changed
}

func (ir *TypedIR) inferHostCallType(
	call *ast.HostCallNode,
) (IRType, bool) {
	argumentTypes := make([]IRType, len(call.Args))
	for index, argument := range call.Args {
		argumentTypes[index] = ir.facts[argument].Type
	}
	return InferNumericHostCallType(call, argumentTypes)
}

// InferNumericHostCallType is the shared representation rule for calls to
// lang.Numbers. Specialized region analyses use the same rule as TypedIR
// instead of maintaining their own operation allowlists.
func InferNumericHostCallType(
	call *ast.HostCallNode,
	argumentTypes []IRType,
) (IRType, bool) {
	if call == nil || call.Method == nil ||
		call.Target == nil || call.Target.Op != ast.OpConst ||
		len(call.Args) != len(argumentTypes) {
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
	allNumeric := len(call.Args) > 0
	hasFloat := false
	for _, argument := range argumentTypes {
		kind := argument.Kind
		if kind != IRInt {
			allInt = false
		}
		if kind != IRInt && kind != IRFloat {
			allNumeric = false
		}
		if kind == IRFloat {
			hasFloat = true
		}
	}
	if !allNumeric {
		return IRType{}, false
	}
	switch strings.ToLower(call.Method.Name()) {
	case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec",
		"add", "uncheckedadd", "minus", "unchecked_minus", "multiply",
		"unchecked_multiply":
		if hasFloat {
			return IRType{
				Kind:   IRFloat,
				GoType: reflect.TypeOf(float64(0)),
			}, true
		}
		return IRType{Kind: IRInt, GoType: reflect.TypeOf(int64(0))}, true
	case "divide":
		if hasFloat {
			return IRType{
				Kind:   IRFloat,
				GoType: reflect.TypeOf(float64(0)),
			}, true
		}
	case "quotient", "remainder":
		if allInt {
			return IRType{
				Kind:   IRInt,
				GoType: reflect.TypeOf(int64(0)),
			}, true
		}
	case "lt", "lte", "gt", "gte", "iszero", "ispos", "isneg":
		if allInt || hasFloat && !allInt {
			return IRType{Kind: IRBool, GoType: reflect.TypeOf(false)}, true
		}
	default:
		return IRType{}, false
	}
	return IRType{}, false
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
		do := node.Sub.(*ast.DoNode)
		facts.Type = ir.facts[do.Ret].Type
		for _, statement := range do.Statements {
			facts.NeverReturns = facts.NeverReturns ||
				ir.facts[statement].NeverReturns
		}
		facts.NeverReturns = facts.NeverReturns ||
			ir.facts[do.Ret].NeverReturns
	case ast.OpLet, ast.OpLoop:
		let := node.Sub.(*ast.LetNode)
		facts.Type = ir.facts[let.Body].Type
		for _, binding := range let.Bindings {
			init := binding.Sub.(*ast.BindingNode).Init
			facts.NeverReturns = facts.NeverReturns ||
				ir.facts[init].NeverReturns
		}
		facts.NeverReturns = facts.NeverReturns ||
			ir.facts[let.Body].NeverReturns
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		facts.Type = irJoinBranchTypes(
			ir.facts[ifNode.Then],
			ir.facts[ifNode.Else],
		)
		facts.NeverReturns = ir.facts[ifNode.Then].NeverReturns &&
			ifNode.Else != nil && ir.facts[ifNode.Else].NeverReturns
	case ast.OpCase:
		caseNode := node.Sub.(*ast.CaseNode)
		facts.Type = ir.caseResultType(caseNode)
		facts.NeverReturns = ir.caseNeverReturns(caseNode)
	case ast.OpThrow:
		facts.NeverReturns = true
		facts.Effects |= IREffectMayThrow
	case ast.OpRecur:
		facts.NeverReturns = true
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
	if signature := ir.resolveCallSignature(invoke); signature != nil {
		facts.Type = signature.Result
		facts.Signature = signature
	}

	if namespace == "clojure.string" {
		switch name {
		case "join", "replace":
			facts.Type = IRType{
				Kind:   IRString,
				GoType: reflect.TypeOf(""),
			}
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
		facts.Type = IRType{Kind: IRAtom, GoType: reflect.TypeOf((*lang.Atom)(nil))}
		facts.Effects |= IREffectAllocate
	case "count":
		facts.Type = IRType{Kind: IRInt, GoType: reflect.TypeOf(int(0))}
	case "str":
		facts.Type = IRType{Kind: IRString, GoType: reflect.TypeOf("")}
	case "subs":
		facts.Type = IRType{Kind: IRString, GoType: reflect.TypeOf("")}
		facts.Effects |= IREffectMayThrow
	case "=":
		facts.Type = IRType{Kind: IRBool, GoType: reflect.TypeOf(false)}
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

func (ir *TypedIR) resolveCallSignature(
	invoke *ast.InvokeNode,
) *IRCallSignature {
	if ir == nil || invoke == nil || invoke.Fn == nil ||
		invoke.Fn.Op != ast.OpVar {
		return nil
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	for i := range ir.callSignatures[vr] {
		signature := &ir.callSignatures[vr][i]
		if len(signature.Params) != len(invoke.Args) {
			continue
		}
		matched := true
		for index, argument := range invoke.Args {
			if !signature.Params[index].Accepts(ir.facts[argument].Type) {
				matched = false
				break
			}
		}
		if matched {
			return signature
		}
	}
	return nil
}

func (ir *TypedIR) refineInvokeType(
	invoke *ast.InvokeNode,
	call IRCall,
) (IRType, bool) {
	if invoke == nil || !call.Known || call.Var == nil ||
		call.Var.Namespace() == nil ||
		call.Var.Namespace().Name().String() != "clojure.core" {
		return IRType{}, false
	}
	switch call.Name {
	case "long":
		if len(invoke.Args) == 1 {
			return IRType{Kind: IRInt, GoType: reflect.TypeOf(int64(0))}, true
		}
	case "mod", "quot", "rem":
		if len(invoke.Args) == 2 &&
			ir.facts[invoke.Args[0]].Type.Kind == IRInt &&
			ir.facts[invoke.Args[1]].Type.Kind == IRInt {
			return IRType{Kind: IRInt, GoType: reflect.TypeOf(int64(0))}, true
		}
	case "inc", "dec":
		if len(invoke.Args) == 1 {
			kind := ir.facts[invoke.Args[0]].Type.Kind
			if kind == IRInt || kind == IRFloat {
				return IRType{Kind: kind, GoType: canonicalIRGoType(kind)}, true
			}
		}
	}
	return IRType{}, false
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
			if node.Op == ast.OpLoop {
				result.OwnedStringParts =
					ir.analyzeOwnedStringParts(binding, i, let)
			}
			if result.OwnedStringParts {
				result.Escape = IRDoesNotEscape
			}
			if node.Op == ast.OpLoop {
				result.OwnedMapMode = ir.analyzeOwnedMap(binding, i, let)
			}
			if result.OwnedMapMode != IROwnedMapNone {
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
	if irNestedFunctionCapturesLocal(body, bindingNode.Name) {
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

// irNestedFunctionCapturesLocal reports whether a local representation would
// cross a synchronous ownership region through a closure or goroutine. Even a
// read-only-looking use is unsafe: the closure may run after a later mutation
// and observe a value that persistent semantics would have snapshotted.
func irNestedFunctionCapturesLocal(
	root *ast.Node,
	target *lang.Symbol,
) bool {
	captured := false
	var walk func(*ast.Node, bool)
	walk = func(node *ast.Node, nested bool) {
		if node == nil || captured {
			return
		}
		if nested && irLocalIs(node, target) {
			captured = true
			return
		}
		nested = nested || node.Op == ast.OpFn ||
			node.Op == ast.OpFnMethod || node.Op == ast.OpGo
		for _, child := range irChildren(node) {
			walk(child, nested)
		}
	}
	walk(root, false)
	return captured
}

func irConstType(value any) IRType {
	typ := reflect.TypeOf(value)
	switch value.(type) {
	case nil:
		return IRType{Kind: IRNil, Nullable: true}
	case bool:
		return IRType{Kind: IRBool, GoType: typ}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return IRType{Kind: IRInt, GoType: typ}
	case float32, float64:
		return IRType{Kind: IRFloat, GoType: typ}
	case string:
		return IRType{Kind: IRString, GoType: typ}
	case lang.Keyword:
		return IRType{Kind: IRKeyword, GoType: typ}
	default:
		return IRType{
			Kind:     IRDynamic,
			Nullable: true,
			GoType:   typ,
		}
	}
}

// ConstantType exposes the shared representation assigned to an AST
// constant for region analyses that also track ownership.
func ConstantType(value any) IRType {
	return irConstType(value)
}

func irJoinTypes(types ...IRType) IRType {
	if len(types) == 0 {
		return IRType{Kind: IRDynamic, Nullable: true}
	}
	result := types[0]
	for _, typ := range types[1:] {
		if result.Kind != typ.Kind {
			return IRType{Kind: IRDynamic, Nullable: true}
		}
		result.Nullable = result.Nullable || typ.Nullable
		if result.GoType != typ.GoType {
			result.GoType = nil
		}
	}
	return result
}

func irJoinBranchTypes(branches ...IRFacts) IRType {
	types := make([]IRType, 0, len(branches))
	for _, branch := range branches {
		if !branch.NeverReturns {
			types = append(types, branch.Type)
		}
	}
	return irJoinTypes(types...)
}

func (ir *TypedIR) caseResultType(node *ast.CaseNode) IRType {
	if node == nil || len(node.Entries) == 0 {
		return IRType{Kind: IRDynamic, Nullable: true}
	}
	branches := make([]IRFacts, 0, len(node.Entries)+1)
	for _, entry := range node.Entries {
		branches = append(branches, ir.facts[entry.ResultExpr])
	}
	if node.Default != nil {
		branches = append(branches, ir.facts[node.Default])
	}
	return irJoinBranchTypes(branches...)
}

func (ir *TypedIR) caseNeverReturns(node *ast.CaseNode) bool {
	if node == nil || len(node.Entries) == 0 {
		return false
	}
	for _, entry := range node.Entries {
		if !ir.facts[entry.ResultExpr].NeverReturns {
			return false
		}
	}
	return node.Default == nil || ir.facts[node.Default].NeverReturns
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

// IRChildren returns the immediate AST inputs used by shared IR analyses.
// Region analyses should use this rather than maintaining incomplete local
// AST walkers.
func IRChildren(node *ast.Node) []*ast.Node {
	return irChildren(node)
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
