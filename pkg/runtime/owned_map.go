package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// ownedMap is an internal mutable representation for a map whose identity is
// proved not to escape a reduction. Nested maps are converted lazily, and the
// entire reachable owned tree is frozen before it becomes observable.
type ownedMap struct {
	transient *lang.TransientMap
	children  []ownedMapChild
	meta      lang.IPersistentMap
}

type ownedMapChild struct {
	key   interface{}
	value *ownedMap
}

type ownedMapUpdateArgs struct {
	count  int
	first  interface{}
	second interface{}
	third  interface{}
	rest   []interface{}
}

func newOwnedMap(value interface{}) *ownedMap {
	if value == nil {
		value = lang.NewMap()
	}
	editable, ok := value.(lang.IEditableCollection)
	if !ok {
		panic(lang.NewIllegalArgumentError("owned map reduction requires an editable map"))
	}
	transient, ok := editable.AsTransient().(*lang.TransientMap)
	if !ok {
		panic(lang.NewIllegalArgumentError("owned map reduction requires a transient map"))
	}
	var meta lang.IPersistentMap
	if valueWithMeta, ok := value.(lang.IMeta); ok {
		meta = valueWithMeta.Meta()
	}
	return &ownedMap{transient: transient, meta: meta}
}

func (m *ownedMap) child(key interface{}) *ownedMap {
	value := m.transient.ValAt(key)
	if child, ok := value.(*ownedMap); ok {
		return child
	}
	child := newOwnedMap(value)
	m.transient.Assoc(key, child)
	m.children = append(m.children, ownedMapChild{key: key, value: child})
	return child
}

func (m *ownedMap) leaf(key interface{}) interface{} {
	value := m.transient.ValAt(key)
	if child, ok := value.(*ownedMap); ok {
		value = child.persistent()
		m.transient.Assoc(key, value)
	}
	return value
}

func (m *ownedMap) assoc(key, value interface{}) {
	m.transient.Assoc(key, value)
}

func (m *ownedMap) persistent() interface{} {
	for _, child := range m.children {
		if m.transient.ValAt(child.key) == child.value {
			m.transient.Assoc(child.key, child.value.persistent())
		}
	}
	result := m.transient.Persistent()
	if m.meta != nil {
		if obj, ok := result.(lang.IObj); ok {
			return obj.WithMeta(m.meta)
		}
	}
	return result
}

// ReduceOwnedMap runs a reduction whose accumulator has been proved to be a
// uniquely owned map. The original persistent initial value remains unchanged,
// and the mutable representation is frozen at the reduction boundary.
func ReduceOwnedMap(
	reduceFn, reducer, initial, collection interface{},
) interface{} {
	result := lang.Apply3(
		reduceFn,
		reducer,
		newOwnedMap(initial),
		collection,
	)
	if owned, ok := result.(*ownedMap); ok {
		return owned.persistent()
	}
	return result
}

// AssocOwnedMap applies one assoc entry to a map whose identity is confined to
// an ownership region. Repeated calls cover assoc's variadic entry list while
// preserving source evaluation order in generated code.
func AssocOwnedMap(target, key, value interface{}) interface{} {
	owned := requireOwnedMap(target)
	owned.assoc(key, value)
	return owned
}

// UpdateOwnedMap3 is the fixed-arity owned-map counterpart of update-in.
func UpdateOwnedMap3(target, keys, updateFn interface{}) interface{} {
	return updateOwnedMap(target, keys, updateFn, ownedMapUpdateArgs{})
}

// UpdateOwnedMap4 is the fixed-arity owned-map counterpart of update-in with
// one extra callback argument.
func UpdateOwnedMap4(target, keys, updateFn, arg interface{}) interface{} {
	return updateOwnedMap(target, keys, updateFn, ownedMapUpdateArgs{
		count: 1,
		first: arg,
	})
}

// UpdateOwnedMapPath2_3 is the fixed two-key-path counterpart of update-in.
// The compiler uses it only when the path vector's exact length is known, so
// dynamic update-in calls retain the ordinary collection-based path.
func UpdateOwnedMapPath2_3(
	target, firstKey, secondKey, updateFn interface{},
) interface{} {
	owned := requireOwnedMap(target)
	child := owned.child(firstKey)
	child.assoc(
		secondKey,
		applyOwnedMapUpdate(
			updateFn,
			child.leaf(secondKey),
			ownedMapUpdateArgs{},
		),
	)
	return owned
}

// UpdateOwnedMapPath2_4 handles the corresponding update-in arity with one
// extra callback argument.
func UpdateOwnedMapPath2_4(
	target, firstKey, secondKey, updateFn, arg interface{},
) interface{} {
	owned := requireOwnedMap(target)
	child := owned.child(firstKey)
	child.assoc(
		secondKey,
		applyOwnedMapUpdate(
			updateFn,
			child.leaf(secondKey),
			ownedMapUpdateArgs{count: 1, first: arg},
		),
	)
	return owned
}

// UpdateOwnedMapPath2Default3 applies a one-default fnil update without
// allocating the short-lived wrapper function.
func UpdateOwnedMapPath2Default3(
	target, firstKey, secondKey, updateFn, fallback interface{},
) interface{} {
	owned := requireOwnedMap(target)
	child := owned.child(firstKey)
	current := child.leaf(secondKey)
	if lang.IsNil(current) {
		current = fallback
	}
	child.assoc(secondKey, lang.Apply1(updateFn, current))
	return owned
}

// UpdateOwnedMapPath2Default4 is the corresponding update with one additional
// callback argument.
func UpdateOwnedMapPath2Default4(
	target, firstKey, secondKey, updateFn, fallback, arg interface{},
) interface{} {
	owned := requireOwnedMap(target)
	child := owned.child(firstKey)
	current := child.leaf(secondKey)
	if lang.IsNil(current) {
		current = fallback
	}
	child.assoc(secondKey, lang.Apply2(updateFn, current, arg))
	return owned
}

// UpdateOwnedMap handles update-in calls with two or more extra callback
// arguments. Fixed arities avoid constructing this slice in the common cases.
func UpdateOwnedMap(
	target, keys, updateFn interface{},
	args ...interface{},
) interface{} {
	fixed := ownedMapUpdateArgs{count: len(args)}
	switch len(args) {
	case 0:
	case 1:
		fixed.first = args[0]
	case 2:
		fixed.first, fixed.second = args[0], args[1]
	case 3:
		fixed.first, fixed.second, fixed.third = args[0], args[1], args[2]
	default:
		fixed.rest = args
	}
	return updateOwnedMap(target, keys, updateFn, fixed)
}

func updateOwnedMap(
	target, keys, updateFn interface{},
	args ownedMapUpdateArgs,
) interface{} {
	owned := requireOwnedMap(target)
	if vector, ok := keys.(lang.IPersistentVector); ok {
		updateOwnedMapVector(owned, vector, 0, updateFn, args)
		return owned
	}
	updateOwnedMapSeq(owned, lang.Seq(keys), updateFn, args)
	return owned
}

func requireOwnedMap(target interface{}) *ownedMap {
	owned, ok := target.(*ownedMap)
	if !ok {
		panic(lang.NewIllegalArgumentError("owned update requires an owned map"))
	}
	return owned
}

func updateOwnedMapVector(
	target *ownedMap,
	keys lang.IPersistentVector,
	index int,
	updateFn interface{},
	args ownedMapUpdateArgs,
) {
	var key interface{}
	if index < keys.Count() {
		key = keys.Nth(index)
	}
	if index+1 < keys.Count() {
		updateOwnedMapVector(target.child(key), keys, index+1, updateFn, args)
		return
	}
	target.assoc(key, applyOwnedMapUpdate(updateFn, target.leaf(key), args))
}

func updateOwnedMapSeq(
	target *ownedMap,
	keys lang.ISeq,
	updateFn interface{},
	args ownedMapUpdateArgs,
) {
	var key interface{}
	var remaining lang.ISeq
	if keys != nil {
		key = keys.First()
		remaining = keys.Next()
	}
	if remaining != nil {
		updateOwnedMapSeq(target.child(key), remaining, updateFn, args)
		return
	}
	target.assoc(key, applyOwnedMapUpdate(updateFn, target.leaf(key), args))
}

func applyOwnedMapUpdate(
	updateFn, current interface{},
	args ownedMapUpdateArgs,
) interface{} {
	switch args.count {
	case 0:
		return lang.Apply1(updateFn, current)
	case 1:
		return lang.Apply2(updateFn, current, args.first)
	case 2:
		return lang.Apply3(updateFn, current, args.first, args.second)
	case 3:
		return lang.Apply4(
			updateFn,
			current,
			args.first,
			args.second,
			args.third,
		)
	default:
		values := make([]interface{}, len(args.rest)+1)
		values[0] = current
		copy(values[1:], args.rest)
		return lang.Apply(updateFn, values)
	}
}
