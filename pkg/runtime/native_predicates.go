package runtime

import "github.com/glojurelang/glojure/pkg/lang"

func installNativeCoreTypePredicates(core *lang.Namespace) {
	predicates := []struct {
		name string
		fn   lang.FnFunc1
	}{
		{"string?", func(value any) any { _, ok := value.(string); return ok }},
		{"char?", func(value any) any { _, ok := value.(lang.Char); return ok }},
		{"map?", func(value any) any { _, ok := value.(lang.IPersistentMap); return ok }},
		{"vector?", func(value any) any { _, ok := value.(lang.IPersistentVector); return ok }},
		{"set?", func(value any) any { _, ok := value.(lang.IPersistentSet); return ok }},
		{"seq?", func(value any) any { _, ok := value.(lang.ISeq); return ok }},
	}
	for _, predicate := range predicates {
		if variable := core.FindInternedVar(lang.NewSymbol(predicate.name)); variable != nil {
			variable.BindRoot(predicate.fn)
		}
	}
}
