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
