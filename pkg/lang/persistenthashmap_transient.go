package lang

// Transient nodes embed their persistent counterpart so lookup, iteration, and
// post-persistent! updates reuse the persistent implementation. Only nodes
// owned by a transient carry an edit token, preserving the compact persistent
// node layout.
type (
	transientBitmapIndexedNode struct {
		BitmapIndexedNode
		edit *transientMapEdit
	}

	transientArrayNode struct {
		ArrayNode
		edit *transientMapEdit
	}

	transientHashCollisionNode struct {
		HashCollisionNode
		edit *transientMapEdit
	}
)

func (b *transientBitmapIndexedNode) assoc(
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	updated := b.BitmapIndexedNode.assoc(shift, hash, key, value, addedLeaf)
	if updated == &b.BitmapIndexedNode {
		return b
	}
	return updated
}

func (b *transientBitmapIndexedNode) without(
	shift uint,
	hash uint32,
	key any,
) Node {
	updated := b.BitmapIndexedNode.without(shift, hash, key)
	if updated == &b.BitmapIndexedNode {
		return b
	}
	return updated
}

func (n *transientArrayNode) assoc(
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	updated := n.ArrayNode.assoc(shift, hash, key, value, addedLeaf)
	if updated == &n.ArrayNode {
		return n
	}
	return updated
}

func (n *transientArrayNode) without(
	shift uint,
	hash uint32,
	key any,
) Node {
	updated := n.ArrayNode.without(shift, hash, key)
	if updated == &n.ArrayNode {
		return n
	}
	return updated
}

func (n *transientHashCollisionNode) assoc(
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	updated := n.HashCollisionNode.assoc(shift, hash, key, value, addedLeaf)
	if updated == &n.HashCollisionNode {
		return n
	}
	return updated
}

func (n *transientHashCollisionNode) without(
	shift uint,
	hash uint32,
	key any,
) Node {
	updated := n.HashCollisionNode.without(shift, hash, key)
	if updated == &n.HashCollisionNode {
		return n
	}
	return updated
}

func (b *BitmapIndexedNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	capacity := len(b.array) + 2
	if capacity < 4 {
		capacity = 4
	}
	array := make([]any, len(b.array), capacity)
	copy(array, b.array)
	return (&transientBitmapIndexedNode{
		BitmapIndexedNode: BitmapIndexedNode{
			bitmap: b.bitmap,
			array:  array,
		},
		edit: edit,
	}).assocTransient(edit, shift, hash, key, value, addedLeaf)
}

func (b *BitmapIndexedNode) withoutTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	removedLeaf *Box,
) Node {
	return (&transientBitmapIndexedNode{
		BitmapIndexedNode: BitmapIndexedNode{
			bitmap: b.bitmap,
			array:  b.array,
		},
	}).withoutTransient(edit, shift, hash, key, removedLeaf)
}

func (n *ArrayNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	array := make([]*nodeSlot, len(n.array))
	copy(array, n.array)
	return (&transientArrayNode{
		ArrayNode: ArrayNode{
			count: n.count,
			array: array,
		},
		edit: edit,
	}).assocTransient(edit, shift, hash, key, value, addedLeaf)
}

func (n *ArrayNode) withoutTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	removedLeaf *Box,
) Node {
	return (&transientArrayNode{
		ArrayNode: ArrayNode{
			count: n.count,
			array: n.array,
		},
	}).withoutTransient(edit, shift, hash, key, removedLeaf)
}

func (n *HashCollisionNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	array := make([]any, len(n.array), len(n.array)+2)
	copy(array, n.array)
	return (&transientHashCollisionNode{
		HashCollisionNode: HashCollisionNode{
			hash:  n.hash,
			count: n.count,
			array: array,
		},
		edit: edit,
	}).assocTransient(edit, shift, hash, key, value, addedLeaf)
}

func (n *HashCollisionNode) withoutTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	removedLeaf *Box,
) Node {
	return (&transientHashCollisionNode{
		HashCollisionNode: HashCollisionNode{
			hash:  n.hash,
			count: n.count,
			array: n.array,
		},
	}).withoutTransient(edit, shift, hash, key, removedLeaf)
}

func (b *transientBitmapIndexedNode) ensureEditable(edit *transientMapEdit) *transientBitmapIndexedNode {
	if b.edit == edit {
		return b
	}
	capacity := len(b.array) + 2
	if capacity < 4 {
		capacity = 4
	}
	array := make([]any, len(b.array), capacity)
	copy(array, b.array)
	return &transientBitmapIndexedNode{
		BitmapIndexedNode: BitmapIndexedNode{
			bitmap: b.bitmap,
			array:  array,
		},
		edit: edit,
	}
}

func (b *transientBitmapIndexedNode) editAndSet(
	edit *transientMapEdit,
	index int,
	value any,
) *transientBitmapIndexedNode {
	editable := b.ensureEditable(edit)
	editable.array[index] = value
	return editable
}

func (b *transientBitmapIndexedNode) editAndSet2(
	edit *transientMapEdit,
	firstIndex int,
	firstValue any,
	secondIndex int,
	secondValue any,
) *transientBitmapIndexedNode {
	editable := b.ensureEditable(edit)
	editable.array[firstIndex] = firstValue
	editable.array[secondIndex] = secondValue
	return editable
}

func (b *transientBitmapIndexedNode) editAndRemovePair(
	edit *transientMapEdit,
	bit int,
	index int,
) Node {
	if b.bitmap == bit {
		return nil
	}
	editable := b.ensureEditable(edit)
	start := 2 * index
	copy(editable.array[start:], editable.array[start+2:])
	last := len(editable.array) - 2
	editable.array[last] = nil
	editable.array[last+1] = nil
	editable.array = editable.array[:last]
	editable.bitmap ^= bit
	return editable
}

func (b *transientBitmapIndexedNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	bit := bitpos(hash, shift)
	index := b.index(bit)
	if b.bitmap&bit != 0 {
		keyOrNil := b.array[2*index]
		valueOrNode := b.array[2*index+1]
		if child, ok := valueOrNode.(Node); ok {
			updated := child.assocTransient(
				edit,
				shift+5,
				hash,
				key,
				value,
				addedLeaf,
			)
			if updated == child {
				return b
			}
			return b.editAndSet(edit, 2*index+1, updated)
		}
		if Equiv(key, keyOrNil) {
			if Identical(value, valueOrNode) {
				return b
			}
			return b.editAndSet(edit, 2*index+1, value)
		}
		addedLeaf.val = addedLeaf
		return b.editAndSet2(
			edit,
			2*index,
			nil,
			2*index+1,
			createNodeTransient(
				edit,
				shift+5,
				keyOrNil,
				valueOrNode,
				hash,
				key,
				value,
			),
		)
	}

	count := bitCount(b.bitmap)
	if count >= 16 {
		nodes := make([]*nodeSlot, 32)
		newIndex := mask(hash, shift)
		nodes[newIndex] = &nodeSlot{
			node: emptyIndexedNode.assocTransient(
				edit,
				shift+5,
				hash,
				key,
				value,
				addedLeaf,
			),
		}
		pair := 0
		for i := 0; i < len(nodes); i++ {
			if (b.bitmap>>i)&1 == 0 {
				continue
			}
			if child, ok := b.array[pair+1].(Node); ok {
				nodes[i] = &nodeSlot{node: child}
			} else {
				existingKey := b.array[pair]
				nodes[i] = &nodeSlot{
					node: emptyIndexedNode.assocTransient(
						edit,
						shift+5,
						HashEq(existingKey),
						existingKey,
						b.array[pair+1],
						addedLeaf,
					),
				}
			}
			pair += 2
		}
		return &transientArrayNode{
			ArrayNode: ArrayNode{
				count: count + 1,
				array: nodes,
			},
			edit: edit,
		}
	}

	editable := b.ensureEditable(edit)
	oldLength := len(editable.array)
	if oldLength+2 > cap(editable.array) {
		capacity := 2 * (count + 4)
		array := make([]any, oldLength, capacity)
		copy(array, editable.array)
		editable.array = array
	}
	editable.array = editable.array[:oldLength+2]
	start := 2 * index
	copy(editable.array[start+2:], editable.array[start:oldLength])
	editable.array[start] = key
	editable.array[start+1] = value
	editable.bitmap |= bit
	addedLeaf.val = addedLeaf
	return editable
}

func (b *transientBitmapIndexedNode) withoutTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	removedLeaf *Box,
) Node {
	bit := bitpos(hash, shift)
	if b.bitmap&bit == 0 {
		return b
	}
	index := b.index(bit)
	keyOrNil := b.array[2*index]
	valueOrNode := b.array[2*index+1]
	if child, ok := valueOrNode.(Node); ok {
		updated := child.withoutTransient(
			edit,
			shift+5,
			hash,
			key,
			removedLeaf,
		)
		if updated == child {
			return b
		}
		if updated != nil {
			return b.editAndSet(edit, 2*index+1, updated)
		}
		return b.editAndRemovePair(edit, bit, index)
	}
	if Equiv(key, keyOrNil) {
		removedLeaf.val = removedLeaf
		return b.editAndRemovePair(edit, bit, index)
	}
	return b
}

func (n *transientArrayNode) ensureEditable(edit *transientMapEdit) *transientArrayNode {
	if n.edit == edit {
		return n
	}
	array := make([]*nodeSlot, len(n.array))
	copy(array, n.array)
	return &transientArrayNode{
		ArrayNode: ArrayNode{
			count: n.count,
			array: array,
		},
		edit: edit,
	}
}

func (n *transientArrayNode) editAndSet(
	edit *transientMapEdit,
	index uint32,
	node Node,
) *transientArrayNode {
	editable := n.ensureEditable(edit)
	if node == nil {
		editable.array[index] = nil
	} else {
		editable.array[index] = &nodeSlot{node: node}
	}
	return editable
}

func (n *transientArrayNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	index := mask(hash, shift)
	slot := n.array[index]
	if slot == nil {
		editable := n.editAndSet(
			edit,
			index,
			emptyIndexedNode.assocTransient(
				edit,
				shift+5,
				hash,
				key,
				value,
				addedLeaf,
			),
		)
		editable.count++
		return editable
	}
	updated := slot.node.assocTransient(
		edit,
		shift+5,
		hash,
		key,
		value,
		addedLeaf,
	)
	if updated == slot.node {
		return n
	}
	return n.editAndSet(edit, index, updated)
}

func (n *transientArrayNode) withoutTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	removedLeaf *Box,
) Node {
	index := mask(hash, shift)
	slot := n.array[index]
	if slot == nil {
		return n
	}
	updated := slot.node.withoutTransient(
		edit,
		shift+5,
		hash,
		key,
		removedLeaf,
	)
	if updated == slot.node {
		return n
	}
	if updated == nil {
		if n.count <= 8 {
			return n.packTransient(edit, index)
		}
		editable := n.editAndSet(edit, index, nil)
		editable.count--
		return editable
	}
	return n.editAndSet(edit, index, updated)
}

func (n *transientArrayNode) packTransient(edit *transientMapEdit, removed uint32) Node {
	array := make([]any, 0, 2*(n.count-1))
	bitmap := 0
	for index, slot := range n.array {
		if uint32(index) == removed || slot == nil {
			continue
		}
		array = append(array, nil, slot.node)
		bitmap |= 1 << index
	}
	return &transientBitmapIndexedNode{
		BitmapIndexedNode: BitmapIndexedNode{
			bitmap: bitmap,
			array:  array,
		},
		edit: edit,
	}
}

func (n *transientHashCollisionNode) ensureEditable(edit *transientMapEdit) *transientHashCollisionNode {
	if n.edit == edit {
		return n
	}
	array := make([]any, len(n.array), len(n.array)+2)
	copy(array, n.array)
	return &transientHashCollisionNode{
		HashCollisionNode: HashCollisionNode{
			hash:  n.hash,
			count: n.count,
			array: array,
		},
		edit: edit,
	}
}

func (n *transientHashCollisionNode) assocTransient(
	edit *transientMapEdit,
	shift uint,
	hash uint32,
	key any,
	value any,
	addedLeaf *Box,
) Node {
	if hash == n.hash {
		index := n.findIndex(key)
		if index >= 0 {
			if Identical(n.array[index+1], value) {
				return n
			}
			editable := n.ensureEditable(edit)
			editable.array[index+1] = value
			return editable
		}
		editable := n.ensureEditable(edit)
		if len(editable.array)+2 > cap(editable.array) {
			array := make([]any, len(editable.array), len(editable.array)+2)
			copy(array, editable.array)
			editable.array = array
		}
		editable.array = append(editable.array, key, value)
		editable.count++
		addedLeaf.val = addedLeaf
		return editable
	}

	array := make([]any, 2, 4)
	array[1] = n
	return (&transientBitmapIndexedNode{
		BitmapIndexedNode: BitmapIndexedNode{
			bitmap: bitpos(n.hash, shift),
			array:  array,
		},
		edit: edit,
	}).assocTransient(edit, shift, hash, key, value, addedLeaf)
}

func (n *transientHashCollisionNode) withoutTransient(
	edit *transientMapEdit,
	_ uint,
	_ uint32,
	key any,
	removedLeaf *Box,
) Node {
	index := n.findIndex(key)
	if index < 0 {
		return n
	}
	removedLeaf.val = removedLeaf
	if n.count == 1 {
		return nil
	}
	editable := n.ensureEditable(edit)
	last := len(editable.array) - 2
	editable.array[index] = editable.array[last]
	editable.array[index+1] = editable.array[last+1]
	editable.array[last] = nil
	editable.array[last+1] = nil
	editable.array = editable.array[:last]
	editable.count--
	return editable
}

func createNodeTransient(
	edit *transientMapEdit,
	shift uint,
	firstKey any,
	firstValue any,
	secondHash uint32,
	secondKey any,
	secondValue any,
) Node {
	firstHash := HashEq(firstKey)
	if firstHash == secondHash {
		array := make([]any, 4, 6)
		array[0] = firstKey
		array[1] = firstValue
		array[2] = secondKey
		array[3] = secondValue
		return &transientHashCollisionNode{
			HashCollisionNode: HashCollisionNode{
				hash:  firstHash,
				count: 2,
				array: array,
			},
			edit: edit,
		}
	}
	addedLeaf := &Box{}
	return emptyIndexedNode.assocTransient(
		edit,
		shift,
		firstHash,
		firstKey,
		firstValue,
		addedLeaf,
	).assocTransient(
		edit,
		shift,
		secondHash,
		secondKey,
		secondValue,
		addedLeaf,
	)
}
