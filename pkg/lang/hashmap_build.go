package lang

// Construction owns its scratch buffers and trie until the map is published.
// Build each node once instead of copying its path for every input pair.
type hashMapBuildEntry struct {
	hash       uint32
	key, value any
}

func buildHashMapNode(entries, scratch []hashMapBuildEntry, shift uint) (Node, int) {
	if len(entries) == 1 {
		storage := &struct {
			BitmapIndexedNode
			pair [2]any
		}{}
		storage.bitmap = bitpos(entries[0].hash, shift)
		storage.pair = [2]any{entries[0].key, entries[0].value}
		storage.array = storage.pair[:]
		return &storage.BitmapIndexedNode, 1
	}
	// All equal hashes share one collision node. Keep the first key object
	// and the last value, matching repeated assoc, including numeric keys.
	sameHash := true
	for _, entry := range entries[1:] {
		if entry.hash != entries[0].hash {
			sameHash = false
			break
		}
	}
	if sameHash {
		pairs := make([]any, 0, 2*len(entries))
		for _, entry := range entries {
			found := false
			for i := 0; i < len(pairs); i += 2 {
				if Equiv(entry.key, pairs[i]) {
					pairs[i+1] = entry.value
					found = true
					break
				}
			}
			if !found {
				pairs = append(pairs, entry.key, entry.value)
			}
		}
		return &HashCollisionNode{hash: entries[0].hash, count: len(pairs) / 2, array: pairs}, len(pairs) / 2
	}
	var sizes, offsets [32]int
	for _, entry := range entries {
		sizes[mask(entry.hash, shift)]++
	}
	occupied, offset := 0, 0
	for i, size := range sizes {
		offsets[i] = offset
		offset += size
		if size != 0 {
			occupied++
		}
	}
	cursors := offsets
	for _, entry := range entries {
		slot := mask(entry.hash, shift)
		scratch[cursors[slot]] = entry
		cursors[slot]++
	}
	count := 0
	if occupied > 16 {
		storage := &struct {
			ArrayNode
			slots    [32]*nodeSlot
			children [32]nodeSlot
		}{}
		storage.count = occupied
		storage.array = storage.slots[:]
		for i, size := range sizes {
			if size == 0 {
				continue
			}
			start := offsets[i]
			node, n := buildHashMapNode(scratch[start:start+size], entries[start:start+size], shift+5)
			storage.children[i].node = node
			storage.slots[i] = &storage.children[i]
			count += n
		}
		return &storage.ArrayNode, count
	}
	node := &BitmapIndexedNode{array: make([]any, 2*occupied)}
	index := 0
	for i, size := range sizes {
		if size == 0 {
			continue
		}
		node.bitmap |= 1 << i
		start := offsets[i]
		if size == 1 {
			entry := scratch[start]
			node.array[index], node.array[index+1] = entry.key, entry.value
			count++
		} else {
			child, n := buildHashMapNode(scratch[start:start+size], entries[start:start+size], shift+5)
			node.array[index+1] = child
			count += n
		}
		index += 2
	}
	return node, count
}
