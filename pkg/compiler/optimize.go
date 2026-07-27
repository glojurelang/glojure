package compiler

import (
	"fmt"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// OptimizationPass is one backend-neutral AST optimization. A pass receives
// the whole tree so it may perform analysis, rewriting, or multiple sweeps.
type OptimizationPass interface {
	Name() string
	Optimize(*ast.Node) (*ast.Node, error)
}

// Optimizer applies AST passes in declaration order. Each pass sees the full
// output of the previous pass.
type Optimizer struct {
	passes []OptimizationPass
}

// OptimizationOptions controls rewrites whose validity depends on compiler
// semantics rather than only on local tree shape.
type OptimizationOptions struct {
	DirectLinking bool
}

// NewOptimizer constructs an optimizer from an ordered pass pipeline.
func NewOptimizer(passes ...OptimizationPass) *Optimizer {
	return &Optimizer{passes: append([]OptimizationPass(nil), passes...)}
}

// NewDefaultOptimizer constructs the standard backend-neutral pass pipeline.
func NewDefaultOptimizer(options OptimizationOptions) *Optimizer {
	passes := []OptimizationPass{foldLiteralNumbersPass{}}
	if options.DirectLinking {
		passes = append(passes, fuseReplaceLastPass{})
	}
	passes = append(passes, lowerKeywordLookupPass{})
	if options.DirectLinking {
		passes = append(passes, lowerAssocPass{})
	}
	return NewOptimizer(passes...)
}

// Optimize applies every pass to root.
func (o *Optimizer) Optimize(root *ast.Node) (*ast.Node, error) {
	if o == nil {
		o = defaultOptimizer()
	}
	var err error
	for _, pass := range o.passes {
		if pass == nil {
			return nil, fmt.Errorf("nil AST optimization pass")
		}
		root, err = pass.Optimize(root)
		if err != nil {
			return nil, fmt.Errorf("AST optimization %q: %w", pass.Name(), err)
		}
		if root == nil {
			return nil, fmt.Errorf(
				"AST optimization %q returned a nil root",
				pass.Name(),
			)
		}
	}
	return root, nil
}

func defaultOptimizer() *Optimizer {
	return NewDefaultOptimizer(OptimizationOptions{})
}

type foldLiteralNumbersPass struct{}

func (foldLiteralNumbersPass) Name() string {
	return "fold-literal-numbers"
}

func (foldLiteralNumbersPass) Optimize(root *ast.Node) (*ast.Node, error) {
	return ast.Transform(root, func(node *ast.Node) (*ast.Node, error) {
		if node.Op != ast.OpHostCall {
			return node, nil
		}
		if value, ok := foldLiteralNumberCall(node.Sub.(*ast.HostCallNode)); ok {
			folded := ast.MakeNode(ast.OpConst, node.Form)
			folded.Env = node.Env
			folded.RawForms = node.RawForms
			folded.IsLiteral = true
			folded.Sub = &ast.ConstNode{
				Type:  classifyType(value),
				Value: value,
			}
			return folded, nil
		}
		return node, nil
	})
}

type fuseReplaceLastPass struct{}

func (fuseReplaceLastPass) Name() string {
	return "fuse-replace-last"
}

func (fuseReplaceLastPass) Optimize(root *ast.Node) (*ast.Node, error) {
	return ast.Transform(root, func(node *ast.Node) (*ast.Node, error) {
		invoke, ok := node.Sub.(*ast.InvokeNode)
		if !ok || len(invoke.Args) != 2 ||
			!isDirectCoreCall(invoke, "conj") {
			return node, nil
		}
		popNode := invoke.Args[0]
		pop, ok := popNode.Sub.(*ast.InvokeNode)
		if !ok || len(pop.Args) != 1 ||
			!isDirectCoreCall(pop, "pop") {
			return node, nil
		}
		fused := ast.MakeNode(ast.OpReplaceLast, node.Form)
		fused.Env = node.Env
		fused.RawForms = node.RawForms
		fused.Sub = &ast.ReplaceLastNode{
			Meta:       invoke.Meta,
			Collection: pop.Args[0],
			Value:      invoke.Args[1],
		}
		return fused, nil
	})
}

func isDirectCoreCall(invoke *ast.InvokeNode, name string) bool {
	if invoke.Fn.Op != ast.OpVar {
		return false
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.IsMacro() || vr.IsDynamic() ||
		lang.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) {
		return false
	}
	ns := vr.Namespace()
	return ns != nil &&
		ns.Name().String() == "clojure.core" &&
		vr.Symbol().String() == name
}

type lowerKeywordLookupPass struct{}

func (lowerKeywordLookupPass) Name() string {
	return "lower-keyword-lookup"
}

func (lowerKeywordLookupPass) Optimize(root *ast.Node) (*ast.Node, error) {
	return ast.Transform(root, func(node *ast.Node) (*ast.Node, error) {
		invoke, ok := node.Sub.(*ast.InvokeNode)
		if !ok || (len(invoke.Args) != 1 && len(invoke.Args) != 2) ||
			invoke.Fn.Op != ast.OpConst {
			return node, nil
		}
		keyword, ok := invoke.Fn.Sub.(*ast.ConstNode).Value.(lang.Keyword)
		if !ok {
			return node, nil
		}
		lowered := ast.MakeNode(ast.OpKeywordLookup, node.Form)
		lowered.Env = node.Env
		lowered.RawForms = node.RawForms
		lookup := &ast.KeywordLookupNode{
			Meta:    invoke.Meta,
			Keyword: keyword,
			Target:  invoke.Args[0],
		}
		if len(invoke.Args) == 2 {
			lookup.Default = invoke.Args[1]
		}
		lowered.Sub = lookup
		return lowered, nil
	})
}

type lowerAssocPass struct{}

func (lowerAssocPass) Name() string {
	return "lower-assoc"
}

func (lowerAssocPass) Optimize(root *ast.Node) (*ast.Node, error) {
	return ast.Transform(root, func(node *ast.Node) (*ast.Node, error) {
		invoke, ok := node.Sub.(*ast.InvokeNode)
		if !ok || len(invoke.Args) < 3 || len(invoke.Args)%2 == 0 ||
			!isDirectCoreCall(invoke, "assoc") {
			return node, nil
		}
		lowered := ast.MakeNode(ast.OpAssoc, node.Form)
		lowered.Env = node.Env
		lowered.RawForms = node.RawForms
		assoc := &ast.AssocNode{
			Meta:   invoke.Meta,
			Target: invoke.Args[0],
			Entries: make(
				[]ast.AssocEntry,
				(len(invoke.Args)-1)/2,
			),
		}
		for i := range assoc.Entries {
			assoc.Entries[i] = ast.AssocEntry{
				Key: invoke.Args[1+i*2],
				Val: invoke.Args[2+i*2],
			}
		}
		lowered.Sub = assoc
		return lowered, nil
	})
}
