package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// OwnedVector is a compiler-only mutable representation for a vector whose
// persistent identity does not escape an optimized synchronous region. It
// recursively copies a proven vector shape on entry and reconstructs ordinary
// persistent vectors at the region boundary.
type OwnedVector struct {
	values   []any
	meta     lang.IPersistentMap
	original *lang.Vector
	dirty    bool
	// ownedChildren marks nested vectors that are not shared with another
	// compiler-owned logical version and can be updated in place.
	ownedChildren map[int]bool
}

// NewOwnedVector converts exactly depth vector levels. It returns false
// without exposing a partial result when the runtime value does not have the
// shape required by the compiled region.
func NewOwnedVector(value any, depth int) (*OwnedVector, bool) {
	if depth < 1 {
		return nil, false
	}
	vector, ok := value.(*lang.Vector)
	if !ok {
		return nil, false
	}
	result := &OwnedVector{
		values:        make([]any, vector.Count()),
		meta:          vector.Meta(),
		original:      vector,
		ownedChildren: make(map[int]bool),
	}
	for index := range result.values {
		item := vector.Nth(index)
		if depth > 1 {
			nested, ok := NewOwnedVector(item, depth-1)
			if !ok {
				return nil, false
			}
			item = nested
			result.ownedChildren[index] = true
		}
		result.values[index] = item
	}
	return result, true
}

func (v *OwnedVector) Count() int {
	return len(v.values)
}

func (v *OwnedVector) Nth(index int) any {
	if index < 0 || index >= len(v.values) {
		panic(lang.NewIndexOutOfBoundsError())
	}
	return v.values[index]
}

func (v *OwnedVector) Nested(index int) *OwnedVector {
	nested, ok := v.Nth(index).(*OwnedVector)
	if !ok {
		panic(lang.NewIllegalArgumentError("owned vector element is not a vector"))
	}
	return nested
}

// NestedSnapshot returns a private copy of a nested vector. Reads through a
// local bound before a later update therefore continue to observe the same
// immutable version that ordinary persistent-vector code would observe.
func (v *OwnedVector) NestedSnapshot(index int) *OwnedVector {
	nested := v.Nested(index)
	return &OwnedVector{
		values: append([]any(nil), nested.values...),
		meta:   nested.meta,
		dirty:  true,
	}
}

func (v *OwnedVector) versionCopy() *OwnedVector {
	return &OwnedVector{
		values:        append([]any(nil), v.values...),
		meta:          v.meta,
		dirty:         true,
		ownedChildren: make(map[int]bool),
	}
}

// AssocCopy creates the next logical persistent-vector version while retaining
// the cheaper compiler-owned representation. Keeping versions distinct
// preserves reads through locals bound before an assoc.
func (v *OwnedVector) AssocCopy(index int, value any) *OwnedVector {
	result := v.versionCopy()
	return result.Assoc(index, value)
}

// AssocIn2Copy path-copies only the two vectors touched by a nested assoc.
// Other nested vectors remain shared until the persistent boundary.
func (v *OwnedVector) AssocIn2Copy(
	outerIndex int,
	innerKey any,
	value any,
) *OwnedVector {
	if outerIndex < 0 || outerIndex >= len(v.values) {
		if outerIndex != len(v.values) {
			panic(lang.NewIndexOutOfBoundsError())
		}
	}
	result := v.versionCopy()
	return result.AssocIn2(outerIndex, innerKey, value)
}

// Assoc applies vector assoc to the current private logical version.
func (v *OwnedVector) Assoc(index int, value any) *OwnedVector {
	if index < 0 || index > len(v.values) {
		panic(lang.NewIndexOutOfBoundsError())
	}
	if index == len(v.values) {
		v.values = append(v.values, value)
		v.dirty = true
		if _, ok := value.(*OwnedVector); ok {
			v.ownedChildren[index] = true
		}
		return v
	}
	v.values[index] = value
	v.dirty = true
	if _, ok := value.(*OwnedVector); ok {
		v.ownedChildren[index] = true
	}
	return v
}

// AssocIn2 updates a nested child in the current logical version. A child
// shared with an earlier version is copied on its first write.
func (v *OwnedVector) AssocIn2(
	outerIndex int,
	innerKey any,
	value any,
) *OwnedVector {
	if outerIndex < 0 || outerIndex > len(v.values) {
		panic(lang.NewIndexOutOfBoundsError())
	}
	if outerIndex == len(v.values) {
		v.values = append(v.values, lang.Assoc(nil, innerKey, value))
		v.dirty = true
		return v
	}
	nested, ok := v.values[outerIndex].(*OwnedVector)
	if !ok {
		v.values[outerIndex] = lang.Assoc(v.values[outerIndex], innerKey, value)
		v.dirty = true
		return v
	}
	if !v.ownedChildren[outerIndex] {
		nested = nested.versionCopy()
		v.values[outerIndex] = nested
		v.ownedChildren[outerIndex] = true
	}
	nested.Assoc(lang.IntCast(innerKey), value)
	v.dirty = true
	return v
}

func (v *OwnedVector) Persistent() *lang.Vector {
	values := make([]any, len(v.values))
	changed := v.dirty || v.original == nil ||
		v.original.Count() != len(v.values)
	for index, value := range v.values {
		if nested, ok := value.(*OwnedVector); ok {
			persistent := nested.Persistent()
			value = persistent
			if !changed {
				original, ok := v.original.Nth(index).(*lang.Vector)
				changed = !ok || original != persistent
			}
		}
		values[index] = value
	}
	if !changed {
		return v.original
	}
	result := lang.NewVector(values...)
	if v.meta != nil {
		result = result.WithMeta(v.meta).(*lang.Vector)
	}
	return result
}
