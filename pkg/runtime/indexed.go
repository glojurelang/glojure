package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// AsLinearIndexed proves that value has a built-in, side-effect-free indexed
// traversal whose Count/Nth order is equivalent to its ordinary reduction
// order. Merely implementing lang.Indexed is insufficient: a user-defined
// implementation may give IReduce different semantics or make Count/Nth
// observable.
func AsLinearIndexed(value any) (lang.Indexed, bool) {
	switch value := value.(type) {
	case *lang.Vector:
		return value, true
	case *lang.SubVector:
		return value, true
	default:
		return nil, false
	}
}

// MustLinearIndexed is used only after a generated entry guard has established
// the same representation fact and propagated it through local aliases.
func MustLinearIndexed(value any) lang.Indexed {
	indexed, ok := AsLinearIndexed(value)
	if !ok {
		panic(lang.NewIllegalArgumentError("expected built-in indexed value"))
	}
	return indexed
}
