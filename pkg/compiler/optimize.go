package compiler

import (
	"fmt"

	"github.com/glojurelang/glojure/pkg/ast"
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

// NewOptimizer constructs an optimizer from an ordered pass pipeline.
func NewOptimizer(passes ...OptimizationPass) *Optimizer {
	return &Optimizer{passes: append([]OptimizationPass(nil), passes...)}
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
	return NewOptimizer(foldLiteralNumbersPass{})
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
