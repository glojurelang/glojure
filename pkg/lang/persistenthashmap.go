package lang

import "errors"

func CreatePersistentHashMap(keyvals any) any {
	return NewPersistentHashMap(seqToSlice(Seq(keyvals))...)
}

type (
	PersistentHashMap struct {
		meta         IPersistentMap
		hash, hasheq uint32

		count int
		root  Node
	}

	BitmapIndexedNode struct {
		bitmap int
		array  []any
	}

	HashCollisionNode struct {
		hash  uint32
		count int
		array []any
	}

	ArrayNode struct {
		count int
		array []Node
	}

	NodeSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		array []any
		i     int
		s     ISeq
	}

	ArrayNodeSeq struct {
		meta  IPersistentMap
		nodes []Node
		i     int
		s     ISeq
	}

	Node interface {
		assoc(shift uint, hash uint32, key any, val any, addedLeaf *Box) Node
		without(shift uint, hash uint32, key any) Node
		find(shift uint, hash uint32, key any) *Pair
		nodeSeq() ISeq
		iter() MapIterator
	}

	MapIterator interface {
		HasNext() bool
		Next() *Pair
	}

	EmptyMapIterator struct {
	}

	Pair struct {
		Key   any
		Value any
	}

	NodeIterator struct {
		array     []any
		i         int
		nextEntry *Pair
		nextIter  MapIterator
	}

	ArrayNodeIterator struct {
		array      []Node
		i          int
		nestedIter MapIterator
	}
)

var (
	_ APersistentMap = (*PersistentHashMap)(nil)
	_ IPersistentMap = (*PersistentHashMap)(nil)
	_ IMeta          = (*PersistentHashMap)(nil)
	_ IObj           = (*PersistentHashMap)(nil)
	_ IFn            = (*PersistentHashMap)(nil)
	_ IReduce        = (*PersistentHashMap)(nil)
	_ IReduceInit    = (*PersistentHashMap)(nil)

	emptyPersistentHashMap = &PersistentHashMap{}

	emptyIndexedNode = &BitmapIndexedNode{}
)

func NewPersistentHashMap(keyvals ...any) IPersistentMap {
	var res Associative = emptyPersistentHashMap
	for i := 0; i < len(keyvals); i += 2 {
		res = res.Assoc(keyvals[i], keyvals[i+1])
	}
	return res.(*PersistentHashMap)
}

func (m *PersistentHashMap) Meta() IPersistentMap {
	return m.meta
}

func (m *PersistentHashMap) WithMeta(meta IPersistentMap) any {
	if m.meta == meta {
		return m
	}
	cpy := *m
	cpy.meta = meta
	return &cpy
}

func (m *PersistentHashMap) Assoc(key, val any) Associative {
	addedLeaf := &Box{}
	var newroot, t Node
	if m.root == nil {
		t = emptyIndexedNode
	} else {
		t = m.root
	}

	newroot = t.assoc(0, HashEq(key), key, val, addedLeaf)
	if newroot == m.root {
		return m
	}
	newcount := m.count
	if addedLeaf.val != nil {
		newcount = m.count + 1
	}
	res := &PersistentHashMap{
		count: newcount,
		root:  newroot,
	}
	res.meta = m.meta
	return res
}

func (m *PersistentHashMap) AssocEx(key, val any) IPersistentMap {
	return apersistentmapAssocEx(m, key, val)
}

func (m *PersistentHashMap) Without(key any) IPersistentMap {
	if m.root == nil {
		return m
	}
	newroot := m.root.without(0, HashEq(key), key)
	if newroot == m.root {
		return m
	}
	res := &PersistentHashMap{
		count: m.count - 1,
		root:  newroot,
	}
	res.meta = m.meta
	return res
}

func (m *PersistentHashMap) EntryAt(key any) IMapEntry {
	if m.root != nil {
		p := m.root.find(0, HashEq(key), key)
		if p != nil {
			return &MapEntry{
				key: p.Key,
				val: p.Value,
			}
		}
	}
	return nil
}

func (m *PersistentHashMap) Count() int {
	return m.count
}

func (m *PersistentHashMap) xxx_counted() {}

func (m *PersistentHashMap) Seq() ISeq {
	if m.root != nil {
		return m.root.nodeSeq()
	}
	return nil
}

func (m *PersistentHashMap) Empty() IPersistentCollection {
	return emptyPersistentHashMap.WithMeta(m.Meta()).(IPersistentCollection)
}

func (m *PersistentHashMap) ValAtDefault(key, notFound any) any {
	if m.root != nil {
		if res := m.root.find(0, HashEq(key), key); res != nil {
			return res.Value
		}
	}
	return notFound
}

func (m *PersistentHashMap) Reduce(f IFn) any {
	if m.Count() == 0 {
		return f.Invoke()
	}
	var res any
	first := true
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		if first {
			res = seq.First()
			first = false
			continue
		}
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (m *PersistentHashMap) ReduceInit(f IFn, init any) any {
	res := init
	for seq := Seq(m); seq != nil; seq = seq.Next() {
		res = f.Invoke(res, seq.First())
		if IsReduced(res) {
			return res.(IDeref).Deref()
		}
	}
	return res
}

func (m *PersistentHashMap) String() string {
	return apersistentmapString(m)
}

func (m *PersistentHashMap) Cons(o any) Conser {
	return apersistentmapCons(m, o)
}

func (m *PersistentHashMap) ContainsKey(k any) bool {
	return apersistentmapContainsKey(m, k)
}

func (m *PersistentHashMap) Equiv(o any) bool {
	return apersistentmapEquiv(m, o)
}

func (m *PersistentHashMap) ValAt(key any) any {
	return m.ValAtDefault(key, nil)
}

func (m *PersistentHashMap) ApplyTo(args ISeq) any {
	return afnApplyTo(m, args)
}

func (m *PersistentHashMap) Invoke(args ...any) any {
	return apersistentmapInvoke(m, args...)
}

func (m *PersistentHashMap) HashEq() uint32 {
	return apersistentmapHashEq(&m.hasheq, m)
}

////////////////////////////////////////////////////////////////////////////////
// BitmapIndexedNode

func (b *BitmapIndexedNode) index(bit int) int {
	return bitCount(b.bitmap & (bit - 1))
}

func (b *BitmapIndexedNode) iter() MapIterator {
	return &NodeIterator{
		array: b.array,
	}
}

func (b *BitmapIndexedNode) assoc(shift uint, hash uint32, key any, val any, addedLeaf *Box) Node {
	bit := bitpos(hash, shift)
	idx := b.index(bit)

	if b.bitmap&bit != 0 {
		keyOrNull := b.array[2*idx]
		valOrNode := b.array[2*idx+1]
		if _, ok := valOrNode.(Node); ok {
			n := valOrNode.(Node).assoc(shift+5, hash, key, val, addedLeaf)
			if n == valOrNode {
				return b
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, n),
			}
		}
		if Equiv(key, keyOrNull) {
			if val == valOrNode {
				return b
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, val),
			}
		}
		addedLeaf.val = addedLeaf
		return &BitmapIndexedNode{
			bitmap: b.bitmap,
			array:  cloneAndSet2(b.array, 2*idx, nil, 2*idx+1, createNode(shift+5, keyOrNull, valOrNode, hash, key, val)),
		}
	} else {
		n := bitCount(b.bitmap)
		if n >= 16 {
			nodes := make([]Node, 32)
			jdx := mask(hash, shift)
			nodes[jdx] = emptyIndexedNode.assoc(shift+5, hash, key, val, addedLeaf)
			j := 0
			var i uint
			for i = 0; i < 32; i++ {
				if (b.bitmap>>i)&1 != 0 {
					if node, ok := b.array[j+1].(Node); ok {
						nodes[i] = node
					} else {
						nodes[i] = emptyIndexedNode.assoc(shift+5, HashEq(b.array[j]), b.array[j], b.array[j+1], addedLeaf)
					}
					j += 2
				}
			}
			return &ArrayNode{
				count: n + 1,
				array: nodes,
			}
		} else {
			newArray := make([]any, 2*(n+1))
			for i := 0; i < 2*idx; i++ {
				newArray[i] = b.array[i]
			}
			newArray[2*idx] = key
			addedLeaf.val = addedLeaf
			newArray[2*idx+1] = val
			for i := 2 * idx; i < 2*n; i++ {
				newArray[i+2] = b.array[i]
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap | bit,
				array:  newArray,
			}
		}
	}
}

func (b *BitmapIndexedNode) without(shift uint, hash uint32, key any) Node {
	bit := bitpos(hash, shift)
	if (b.bitmap & bit) == 0 {
		return b
	}
	idx := b.index(bit)
	keyOrNull := b.array[2*idx]
	valOrNode := b.array[2*idx+1]
	if _, ok := valOrNode.(Node); ok {
		n := valOrNode.(Node).without(shift+5, hash, key)
		if n == valOrNode {
			return b
		}
		if n != nil {
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, n),
			}
		}
		if b.bitmap == bit {
			return nil
		}
		return &BitmapIndexedNode{
			bitmap: b.bitmap ^ bit,
			array:  removePair(b.array, idx),
		}
	}
	if Equiv(key, keyOrNull) {
		return &BitmapIndexedNode{
			bitmap: b.bitmap ^ bit,
			array:  removePair(b.array, idx),
		}
	}
	return b
}

func (b *BitmapIndexedNode) find(shift uint, hash uint32, key any) *Pair {
	bit := bitpos(hash, shift)
	if (b.bitmap & bit) == 0 {
		return nil
	}
	idx := b.index(bit)
	keyOrNull := b.array[2*idx]
	valOrNode := b.array[2*idx+1]
	if _, ok := valOrNode.(Node); ok {
		return valOrNode.(Node).find(shift+5, hash, key)
	}
	if Equiv(key, keyOrNull) {
		return &Pair{
			Key:   keyOrNull,
			Value: valOrNode,
		}
	}
	return nil
}

func (b *BitmapIndexedNode) nodeSeq() ISeq {
	return newNodeSeq(b.array, 0, nil)
}

////////////////////////////////////////////////////////////////////////////////
// NodeSeq

func newNodeSeq(array []any, i int, s ISeq) ISeq {
	if s != nil {
		return &NodeSeq{
			array: array,
			i:     i,
			s:     s,
		}
	}
	for j := i; j < len(array); j += 2 {
		switch node := array[j+1].(type) {
		case Node:
			nodeSeq := node.nodeSeq()
			if nodeSeq != nil {
				return &NodeSeq{
					array: array,
					i:     j + 2,
					s:     nodeSeq,
				}
			}
		default:
			return &NodeSeq{
				array: array,
				i:     j,
			}
		}
	}
	return nil
}

func (n *NodeSeq) Meta() IPersistentMap {
	return n.meta
}

func (s *NodeSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	res := *s
	res.meta = meta
	return &res
}

func (s *NodeSeq) String() string {
	return aseqString(s)
}

func (s *NodeSeq) Count() int {
	return aseqCount(s)
}

func (s *NodeSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *NodeSeq) Equals(obj any) bool {
	return aseqEquals(s, obj)
}

func (s *NodeSeq) Equiv(obj any) bool {
	return aseqEquiv(s, obj)
}

func (s *NodeSeq) Seq() ISeq {
	return s
}

func (s *NodeSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *NodeSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *NodeSeq) First() any {
	if s.s != nil {
		return s.s.First()
	}
	return NewMapEntry(s.array[s.i], s.array[s.i+1])
}

func (s *NodeSeq) Next() ISeq {
	var res ISeq
	if s.s != nil {
		next := s.s.Next()
		res = newNodeSeq(s.array, s.i, next)
	} else {
		res = newNodeSeq(s.array, s.i+2, nil)
	}
	return res
}

func (s *NodeSeq) More() ISeq {
	n := s.Next()
	if n == nil {
		return emptyList
	}
	return n
}

func (s *NodeSeq) Cons(obj any) Conser {
	if s.s == nil {
		return NewCons(obj, nil)
	}
	return NewCons(obj, s)
}

func (s *NodeSeq) xxx_sequential() {}

////////////////////////////////////////////////////////////////////////////////
// NodeIterator

func (iter *NodeIterator) advance() bool {
	for iter.i < len(iter.array) {
		key := iter.array[iter.i]
		nodeOrVal := iter.array[iter.i+1]
		iter.i += 2
		if key != nil {
			iter.nextEntry = &Pair{Key: key, Value: nodeOrVal}
			return true
		} else if nodeOrVal != nil {
			iter1 := nodeOrVal.(Node).iter()
			if iter1 != nil && iter1.HasNext() {
				iter.nextIter = iter1
				return true
			}
		}
	}
	return false
}

func (iter *NodeIterator) HasNext() bool {
	if iter.nextEntry != nil || iter.nextIter != nil {
		return true
	}
	return iter.advance()
}

func (iter *NodeIterator) Next() *Pair {
	ret := iter.nextEntry
	if ret != nil {
		iter.nextEntry = nil
		return ret
	} else if iter.nextIter != nil {
		ret := iter.nextIter.Next()
		if !iter.nextIter.HasNext() {
			iter.nextIter = nil
		}
		return ret
	} else if iter.advance() {
		return iter.Next()
	}
	panic(newIteratorError())
}

////////////////////////////////////////////////////////////////////////////////
// ArrayNode

func (n *ArrayNode) iter() MapIterator {
	return &ArrayNodeIterator{
		array: n.array,
	}
}

func (n *ArrayNode) assoc(shift uint, hash uint32, key any, val any, addedLeaf *Box) Node {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return &ArrayNode{
			count: n.count + 1,
			array: cloneAndSetNode(n.array, int(idx), emptyIndexedNode.assoc(shift+5, hash, key, val, addedLeaf)),
		}
	}
	nn := node.assoc(shift+5, hash, key, val, addedLeaf)
	if nn == node {
		return n
	}
	return &ArrayNode{
		count: n.count,
		array: cloneAndSetNode(n.array, int(idx), nn),
	}
}

func (n *ArrayNode) without(shift uint, hash uint32, key any) Node {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return n
	}
	nn := node.without(shift+5, hash, key)
	if nn == node {
		return n
	}
	if nn == nil {
		if n.count <= 8 {
			return n.pack(uint(idx))
		}
		return &ArrayNode{
			count: n.count - 1,
			array: cloneAndSetNode(n.array, int(idx), nn),
		}
	} else {
		return &ArrayNode{
			count: n.count,
			array: cloneAndSetNode(n.array, int(idx), nn),
		}
	}
}

func (n *ArrayNode) find(shift uint, hash uint32, key any) *Pair {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return nil
	}
	return node.find(shift+5, hash, key)
}

func (n *ArrayNode) nodeSeq() ISeq {
	return newArrayNodeSeq(n.array, 0, nil)
}

func (n *ArrayNode) pack(idx uint) Node {
	newArray := make([]any, 2*(n.count-1))
	j := 1
	bitmap := 0
	var i uint
	for i = 0; i < idx; i++ {
		if n.array[i] != nil {
			newArray[j] = n.array[i]
			bitmap |= 1 << i
			j += 2
		}
	}
	for i = idx + 1; i < uint(len(n.array)); i++ {
		if n.array[i] != nil {
			newArray[j] = n.array[i]
			bitmap |= 1 << i
			j += 2
		}
	}
	return &BitmapIndexedNode{
		bitmap: bitmap,
		array:  newArray,
	}
}

////////////////////////////////////////////////////////////////////////////////
// HashCollisionNode

func (n *HashCollisionNode) findIndex(key any) int {
	for i := 0; i < 2*n.count; i += 2 {
		if Equiv(key, n.array[i]) {
			return i
		}
	}
	return -1
}

func (n *HashCollisionNode) iter() MapIterator {
	return &NodeIterator{
		array: n.array,
	}
}

func (n *HashCollisionNode) assoc(shift uint, hash uint32, key any, val any, addedLeaf *Box) Node {
	if hash == n.hash {
		idx := n.findIndex(key)
		if idx != -1 {
			if n.array[idx+1] == val {
				return n
			}
			return &HashCollisionNode{
				hash:  hash,
				count: n.count,
				array: cloneAndSet(n.array, idx+1, val),
			}
		}
		newArray := make([]any, 2*(n.count+1))
		for i := 0; i < 2*n.count; i++ {
			newArray[i] = n.array[i]
		}
		newArray[2*n.count] = key
		newArray[2*n.count+1] = val
		addedLeaf.val = addedLeaf
		return &HashCollisionNode{
			hash:  hash,
			count: n.count + 1,
			array: newArray,
		}
	}
	return (&BitmapIndexedNode{
		bitmap: bitpos(n.hash, shift),
		array:  []any{nil, n},
	}).assoc(shift, hash, key, val, addedLeaf)
}

func (n *HashCollisionNode) without(shift uint, hash uint32, key any) Node {
	idx := n.findIndex(key)
	if idx == -1 {
		return n
	}
	if n.count == 1 {
		return nil
	}
	return &HashCollisionNode{
		hash:  hash,
		count: n.count - 1,
		array: removePair(n.array, idx/2),
	}
}

func (n *HashCollisionNode) find(shift uint, hash uint32, key any) *Pair {
	idx := n.findIndex(key)
	if idx == -1 {
		return nil
	}
	return &Pair{
		Key:   n.array[idx],
		Value: n.array[idx+1],
	}
}

func (n *HashCollisionNode) nodeSeq() ISeq {
	return newNodeSeq(n.array, 0, nil)
}

////////////////////////////////////////////////////////////////////////////////
// ArrayNodeSeq

func newArrayNodeSeq(nodes []Node, i int, s ISeq) ISeq {
	if s != nil {
		return &ArrayNodeSeq{
			nodes: nodes,
			i:     i,
			s:     s,
		}
	}
	for j := i; j < len(nodes); j++ {
		if nodes[j] != nil {
			ns := nodes[j].nodeSeq()
			if ns != nil {
				return &ArrayNodeSeq{
					nodes: nodes,
					i:     j + 1,
					s:     ns,
				}
			}
		}
	}
	return nil
}

func (s *ArrayNodeSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *ArrayNodeSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	res := *s
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (s *ArrayNodeSeq) String() string {
	return aseqString(s)
}

func (s *ArrayNodeSeq) Seq() ISeq {
	return s
}

func (s *ArrayNodeSeq) First() any {
	return s.s.First()
}

func (s *ArrayNodeSeq) Next() ISeq {
	next := s.s.Next()
	res := newArrayNodeSeq(s.nodes, s.i, next)
	return res
}

func (s *ArrayNodeSeq) More() ISeq {
	return aseqMore(s)
}

func (s *ArrayNodeSeq) Cons(obj any) Conser {
	return aseqCons(s, obj)
}

func (s *ArrayNodeSeq) Count() int {
	panic("Not supported")
}

func (s *ArrayNodeSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *ArrayNodeSeq) Equiv(obj any) bool {
	panic("Not supported")
}

func (s *ArrayNodeSeq) Equals(obj any) bool {
	panic("Not supported")
}

func (s *ArrayNodeSeq) Hash() uint32 {
	return hashOrdered(s)
}

func (s *ArrayNodeSeq) HashEq() uint32 {
	panic("Not supported")
}

func (s *ArrayNodeSeq) xxx_sequential() {}

////////////////////////////////////////////////////////////////////////////////
// ArrayNodeIterator

func (iter *ArrayNodeIterator) HasNext() bool {
	for {
		if iter.nestedIter != nil {
			if iter.nestedIter.HasNext() {
				return true
			} else {
				iter.nestedIter = nil
			}
		}
		if iter.i < len(iter.array) {
			node := iter.array[iter.i]
			iter.i++
			if node != nil {
				iter.nestedIter = node.iter()
			}
		} else {
			return false
		}
	}
}

func (iter *ArrayNodeIterator) Next() *Pair {
	if iter.HasNext() {
		return iter.nestedIter.Next()
	}
	panic(newIteratorError())
}

////////////////////////////////////////////////////////////////////////////////
// utils

func bitCount(n int) int {
	var count int
	for n != 0 {
		count++
		n &= n - 1
	}
	return count
}

func mask(hash uint32, shift uint) uint32 {
	return (hash >> shift) & 0x01f
}

func bitpos(hash uint32, shift uint) int {
	return 1 << mask(hash, shift)
}

func bitAt(idx int) int {
	return 1 << idx
}

func cloneAndSet(array []any, i int, a any) []any {
	res := clone(array)
	res[i] = a
	return res
}

func cloneAndSet2(array []any, i int, a any, j int, b any) []any {
	res := clone(array)
	res[i] = a
	res[j] = b
	return res
}

func cloneAndSetNode(array []Node, i int, a Node) []Node {
	res := make([]Node, len(array), cap(array))
	copy(res, array)
	res[i] = a
	return res
}

func createNode(shift uint, key1 any, val1 any, key2hash uint32, key2 any, val2 any) Node {
	key1hash := HashEq(key1)
	if key1hash == key2hash {
		return &HashCollisionNode{
			hash:  key1hash,
			count: 2,
			array: []any{key1, val1, key2, val2},
		}
	}
	addedLeaf := &Box{}
	return emptyIndexedNode.assoc(shift, key1hash, key1, val1, addedLeaf).assoc(shift, key2hash, key2, val2, addedLeaf)
}

func removePair(array []any, n int) []any {
	newArray := make([]any, len(array)-2)
	for i := 0; i < 2*n; i++ {
		newArray[i] = array[i]
	}
	for i := 2 * (n + 1); i < len(array); i++ {
		newArray[i-2] = array[i]
	}
	return newArray
}

func clone(s []any) []any {
	result := make([]any, len(s), cap(s))
	copy(result, s)
	return result
}

func newIteratorError() error {
	return errors.New("iterator reached the end of collection")
}
