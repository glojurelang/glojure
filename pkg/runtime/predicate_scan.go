package runtime

import "github.com/glojurelang/glojure/pkg/lang"

func installNativePredicateScans(core *lang.Namespace) {
	for _, name := range []string{"some", "every?", "not-any?", "not-every?"} {
		mode := name
		installFixedArityCoreFunction(core, name, 2, lang.FnFunc2(func(pred, coll any) any {
			truthy := mode == "some" || mode == "not-any?"
			value, found := scanPredicate(pred, coll, truthy)
			switch mode {
			case "some":
				return value
			case "not-every?":
				return found
			default:
				return !found
			}
		}))
	}
}

// Built-in vectors have immutable indexed storage. Other inputs retain the
// ordinary element-at-a-time sequence traversal and predicate short circuit.
func scanPredicate(pred, coll any, truthy bool) (any, bool) {
	var indexed lang.Indexed
	switch coll.(type) {
	case *lang.Vector, *lang.MapEntry:
		indexed = coll.(lang.Indexed)
	}
	if indexed != nil {
		for i := 0; i < indexed.Count(); i++ {
			value := lang.Apply1(pred, indexed.Nth(i))
			if lang.IsTruthy(value) == truthy {
				return value, true
			}
		}
	} else {
		for seq := lang.Seq(coll); seq != nil; seq = lang.Seq(seq.Next()) {
			value := lang.Apply1(pred, seq.First())
			if lang.IsTruthy(value) == truthy {
				return value, true
			}
		}
	}
	return nil, false
}
