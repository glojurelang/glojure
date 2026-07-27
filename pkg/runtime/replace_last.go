package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// replaceLastPlan preserves the evaluation order of
// (conj (pop collection) value): preparing the plan performs and validates the
// pop before value is evaluated.
type replaceLastPlan struct {
	vector *lang.Vector
	popped any
}

// PrepareReplaceLast prepares the collection side of a fused pop/conj.
func PrepareReplaceLast(collection any) replaceLastPlan {
	if vector, ok := collection.(*lang.Vector); ok {
		if vector.Count() == 0 {
			vector.Pop()
		}
		return replaceLastPlan{vector: vector}
	}
	return replaceLastPlan{popped: RT.Pop(collection)}
}

// Finish installs value after the collection's pop has been validated.
func (p replaceLastPlan) Finish(value any) any {
	if p.vector != nil {
		return p.vector.ReplaceLast(value)
	}
	return lang.ConjAny(p.popped, value)
}
