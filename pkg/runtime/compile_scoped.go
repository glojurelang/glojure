package runtime

import (
	"sort"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type scopedEvalBinding struct {
	name *lang.Symbol
	slot localSlot
}

// compileScopedAST materializes the current lexical slots at a boundary that
// needs the full evaluator. The surrounding expressions stay compiled, and
// closures capture values instead of mutable function or loop frame slots.
func (c threadedEvalCompiler) compileScopedAST(node *ast.Node) evalFn {
	bindings := make([]scopedEvalBinding, 0, len(c.localSlots))
	for name, slot := range c.localSlots {
		bindings = append(bindings, scopedEvalBinding{name: name, slot: slot})
	}
	// Later let slots shadow earlier bindings with the same printed name.
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].slot.kind != bindings[j].slot.kind {
			return bindings[i].slot.kind < bindings[j].slot.kind
		}
		return bindings[i].slot.index < bindings[j].slot.index
	})
	return func(env *environment) (interface{}, error) {
		scoped := env.PushScope().(*environment)
		for _, binding := range bindings {
			slot := binding.slot
			var value interface{}
			switch slot.kind {
			case loopLocalSlot:
				value = env.loopFrame.args[slot.index]
			case letLocalSlot:
				value = env.evalFrame.args[slot.index]
			default:
				value = env.fnFrame.args[slot.index]
			}
			scoped.BindLocal(binding.name, value)
		}
		return scoped.EvalAST(node)
	}
}
