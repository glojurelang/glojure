//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
)

func (g *Generator) generateAOTPartitionFold(invoke *ast.InvokeNode) (string, bool) {
	modes := []struct {
		name, mode string
		arity      int
	}{
		{"reduce", "runtime.PartitionFoldReduce", 3},
		{"some", "runtime.PartitionFoldSome", 2},
		{"every?", "runtime.PartitionFoldEvery", 2},
		{"not-any?", "runtime.PartitionFoldNotAny", 2},
		{"not-every?", "runtime.PartitionFoldNotEvery", 2},
	}
	mode := ""
	for _, candidate := range modes {
		if len(invoke.Args) == candidate.arity && isCoreVarNode(invoke.Fn, candidate.name) {
			mode = candidate.mode
			break
		}
	}
	if mode == "" {
		return "", false
	}
	source := invoke.Args[len(invoke.Args)-1]
	if source.Op != ast.OpInvoke {
		return "", false
	}
	partition := source.Sub.(*ast.InvokeNode)
	if !isCoreVarNode(partition.Fn, "partition") || len(partition.Args) != 2 {
		return "", false
	}
	size, ok := int64ConstValue(partition.Args[0])
	if !ok || size <= 0 {
		return "", false
	}
	// The outer function's earlier arguments precede the partition function
	// lookup. Only move its root check past expressions that cannot redefine
	// functions, preserving the original evaluation and binding order.
	for _, arg := range invoke.Args[:len(invoke.Args)-1] {
		if !purePartitionFoldArgument(arg) {
			return "", false
		}
	}
	guards := make([]string, 0, 2)
	for _, node := range []*ast.Node{invoke.Fn, partition.Fn} {
		vr := node.Sub.(*ast.VarNode).Var
		id := g.allocVarVar(vr.Namespace().Name().String(), vr.Symbol().String())
		if g.currentValueInit != nil && id != g.currentValueInit.name {
			g.currentValueInit.deps[id] = struct{}{}
		}
		guards = append(guards, "runtime.IsDefaultCoreVar("+id+")")
	}
	result := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("if %s {\n", strings.Join(guards, " && "))
	callback := g.generateASTNode(invoke.Args[0])
	initial := "nil"
	if len(invoke.Args) == 3 {
		initial = g.generateASTNode(invoke.Args[1])
	}
	input := g.generateASTNode(partition.Args[1])
	g.writef("%s = runtime.FoldPartition(%s, %s, %s, %d, %s)\n",
		result, mode, callback, initial, size, input)
	g.writef("} else {\n")
	fallback := g.generateInvokeDefault(invoke)
	g.writef("%s = %s\n", result, fallback)
	g.writef("}\n")
	return result, true
}

func purePartitionFoldArgument(node *ast.Node) bool {
	switch node.Op {
	case ast.OpLocal, ast.OpVar, ast.OpFn:
		return true
	default:
		_, ok := constantCollectionValue(node)
		return ok
	}
}
