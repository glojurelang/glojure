package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	chunkBits  = 5
	nodeSize   = 1 << chunkBits
	tailMaxLen = nodeSize
	chunkMask  = nodeSize - 1
)

// Vector is a persistent sequential container for arbitrary values. It supports
// O(1) lookup by index, modification by index, and insertion and removal
// operations at the end. Being a persistent variant of the data structure, it
// is immutable, and provides O(1) operations to create modified versions of the
// vector that shares the underlying data structure, making it suitable for
// concurrent access. The empty value is a valid empty vector.
type Vector interface {
	json.Marshaler
	// Len returns the length of the vector.
	Len() int
	// Index returns the i-th element of the vector, if it exists. The second
	// return value indicates whether the element exists.
	Index(i int) (interface{}, bool)
	// Assoc returns an almost identical Vector, with the i-th element
	// replaced. If the index is smaller than 0 or greater than the length of
	// the vector, it returns nil. If the index is equal to the size of the
	// vector, it is equivalent to Conj.
	Assoc(i int, val interface{}) Vector
	// Conj returns an almost identical Vector, with an additional element
	// appended to the end.
	Conj(val interface{}) Vector
	// Pop returns an almost identical Vector, with the last element removed. It
	// returns nil if the vector is already empty.
	Pop() Vector
	// SubVector returns a subvector containing the elements from i up to but
	// not including j.
	SubVector(i, j int) Vector
	// Iterator returns an iterator over the vector.
	Iterator() Iterator
}

// Iterator is an iterator over vector elements. It can be used like this:
//
//	for it := v.Iterator(); it.HasElem(); it.Next() {
//	    elem := it.Elem()
//	    // do something with elem...
//	}
type Iterator interface {
	// Elem returns the element at the current position.
	Elem() interface{}
	// HasElem returns whether the iterator is pointing to an element.
	HasElem() bool
	// Next moves the iterator to the next position.
	Next()
}

type Persistent struct {
	count int
	// height of the tree structure, defined to be 0 when root is a leaf.
	height    uint
	root      node
	tail      *tailBase
	tailDelta *tailEntry
}

// tailBase is the immutable slice descriptor shared by persistent vector
// versions. Keeping one pointer in Persistent mirrors Clojure's tail-array
// reference instead of copying a three-word Go slice header into every value.
type tailBase []interface{}

// tailEntry records an immutable append after the tail's base slice. Keeping
// at most 32 linked deltas avoids copying the whole persistent-vector tail on
// every Conj while retaining a compact slice for vectors built in bulk.
type tailEntry struct {
	value interface{}
	prev  *tailEntry
}

// TailStorage reserves space for one immutable tail append. It lets wrappers
// that embed Persistent co-allocate their newest tail entry with the wrapper
// while keeping the tail representation private to this package.
type TailStorage struct {
	entry tailEntry
}

// Empty is an empty Vector.
var Empty Vector = &Persistent{}

// node is a node in the vector tree. It is always of the size nodeSize.
type node *[nodeSize]interface{}

func newNode() node {
	return node(&[nodeSize]interface{}{})
}

func clone(n node) node {
	a := *n
	return node(&a)
}

// Count returns the number of elements in a Vector.
func (v *Persistent) Len() int {
	return v.count
}

// treeSize returns the number of elements stored in the tree (as opposed to the
// tail).
func (v *Persistent) treeSize() int {
	if v.count < tailMaxLen {
		return 0
	}
	return ((v.count - 1) >> chunkBits) << chunkBits
}

func (v *Persistent) Index(i int) (interface{}, bool) {
	if i < 0 || i >= v.count {
		return nil, false
	}

	// The following is very similar to sliceFor, but is implemented separately
	// to avoid unnecessary copying.
	if i >= v.treeSize() {
		return v.tailAt(i - v.treeSize()), true
	}
	n := v.root
	for shift := v.height * chunkBits; shift > 0; shift -= chunkBits {
		n = n[(i>>shift)&chunkMask].(node)
	}
	return n[i&chunkMask], true
}

// sliceFor returns the slice where the i-th element is stored. The index must
// be in bound.
func (v *Persistent) sliceFor(i int) []interface{} {
	if i >= v.treeSize() {
		return v.tailSlice()
	}
	n := v.root
	for shift := v.height * chunkBits; shift > 0; shift -= chunkBits {
		n = n[(i>>shift)&chunkMask].(node)
	}
	return n[:]
}

func (v *Persistent) tailLen() int {
	return v.count - v.treeSize()
}

func newTailBase(values []interface{}) *tailBase {
	if len(values) == 0 {
		return nil
	}
	base := tailBase(values)
	return &base
}

func (v *Persistent) baseTail() []interface{} {
	if v.tail == nil {
		return nil
	}
	return []interface{}(*v.tail)
}

func (v *Persistent) tailAt(i int) interface{} {
	base := v.baseTail()
	if i < len(base) {
		return base[i]
	}
	entry := v.tailDelta
	deltaLen := v.tailLen() - len(base)
	for steps := deltaLen - 1 - (i - len(base)); steps > 0; steps-- {
		entry = entry.prev
	}
	return entry.value
}

func (v *Persistent) tailSlice() []interface{} {
	result := make([]interface{}, v.tailLen())
	base := v.baseTail()
	copy(result, base)
	entry := v.tailDelta
	for i := len(result) - 1; i >= len(base); i-- {
		result[i] = entry.value
		entry = entry.prev
	}
	return result
}

func (v *Persistent) tailNode() node {
	var result [nodeSize]interface{}
	base := v.baseTail()
	copy(result[:], base)
	entry := v.tailDelta
	for i := v.tailLen() - 1; i >= len(base); i-- {
		result[i] = entry.value
		entry = entry.prev
	}
	return &result
}

func (v *Persistent) Assoc(i int, val interface{}) Vector {
	result, ok := v.AssocValue(i, val)
	if !ok {
		return nil
	}
	return &result
}

// AssocValue returns updated persistent state by value, avoiding a wrapper
// allocation for callers that embed Persistent directly.
func (v Persistent) AssocValue(i int, val interface{}) (Persistent, bool) {
	if i < 0 || i > v.count {
		return Persistent{}, false
	} else if i == v.count {
		return v.ConjValue(val), true
	}
	if i >= v.treeSize() {
		newTail := v.tailSlice()
		newTail[i&chunkMask] = val
		return Persistent{
			count:  v.count,
			height: v.height,
			root:   v.root,
			tail:   newTailBase(newTail),
		}, true
	}
	return Persistent{
		count:     v.count,
		height:    v.height,
		root:      doAssoc(v.height, v.root, i, val),
		tail:      v.tail,
		tailDelta: v.tailDelta,
	}, true
}

// ReplaceLastValueInto returns persistent state with its final value replaced.
// When the tail ends in a delta, storage holds the replacement delta so the
// operation does not have to materialize and copy the tail.
func (v Persistent) ReplaceLastValueInto(
	val interface{},
	storage *TailStorage,
) (Persistent, bool) {
	if v.count == 0 {
		return Persistent{}, false
	}
	if v.tailDelta != nil {
		storage.entry = tailEntry{
			value: val,
			prev:  v.tailDelta.prev,
		}
		v.tailDelta = &storage.entry
		return v, true
	}
	tail := append([]interface{}(nil), v.baseTail()...)
	tail[len(tail)-1] = val
	v.tail = newTailBase(tail)
	return v, true
}

// doAssoc returns an almost identical tree, with the i-th element replaced by
// val.
func doAssoc(height uint, n node, i int, val interface{}) node {
	m := clone(n)
	if height == 0 {
		m[i&chunkMask] = val
	} else {
		sub := (i >> (height * chunkBits)) & chunkMask
		m[sub] = doAssoc(height-1, m[sub].(node), i, val)
	}
	return m
}

func (v *Persistent) Conj(val interface{}) Vector {
	result := v.ConjValue(val)
	return &result
}

// ConjValue returns updated persistent state by value, avoiding a wrapper
// allocation for callers that embed Persistent directly.
func (v Persistent) ConjValue(val interface{}) Persistent {
	storage := &TailStorage{}
	return v.ConjValueInto(val, storage)
}

// ConjValueInto returns updated persistent state using caller-owned storage
// for the newest tail entry. The storage must remain alive as long as the
// returned vector or a vector derived from it remains reachable.
func (v Persistent) ConjValueInto(val interface{}, storage *TailStorage) Persistent {
	storage.entry = tailEntry{
		value: val,
		prev:  v.tailDelta,
	}

	// Room in tail?
	if v.count-v.treeSize() < tailMaxLen {
		return Persistent{
			count:     v.count + 1,
			height:    v.height,
			root:      v.root,
			tail:      v.tail,
			tailDelta: &storage.entry,
		}
	}
	// Full tail; push into tree.
	tailNode := v.tailNode()
	newHeight := v.height
	var newRoot node

	// Overflow root?
	if (v.count >> chunkBits) > (1 << (v.height * chunkBits)) {
		newRoot = newNode()
		newRoot[0] = v.root
		newRoot[1] = newPath(v.height, tailNode)
		newHeight++
	} else {
		newRoot = v.pushTail(v.height, v.root, tailNode)
	}
	// The old tail now lives in the tree, so the first entry in the new tail
	// must not retain its delta chain.
	storage.entry.prev = nil
	return Persistent{
		count:     v.count + 1,
		height:    newHeight,
		root:      newRoot,
		tailDelta: &storage.entry,
	}
}

// pushTail returns a tree with tail appended.
func (v *Persistent) pushTail(height uint, n node, tail node) node {
	if height == 0 {
		return tail
	}
	idx := ((v.count - 1) >> (height * chunkBits)) & chunkMask
	m := clone(n)
	child := n[idx]
	if child == nil {
		m[idx] = newPath(height-1, tail)
	} else {
		m[idx] = v.pushTail(height-1, child.(node), tail)
	}
	return m
}

// newPath creates a left-branching tree of specified height and leaf.
func newPath(height uint, leaf node) node {
	if height == 0 {
		return leaf
	}
	ret := newNode()
	ret[0] = newPath(height-1, leaf)
	return ret
}

func (v *Persistent) Pop() Vector {
	result, ok := v.PopValue()
	if !ok {
		return nil
	}
	if result.count == 0 {
		return Empty
	}
	return &result
}

// PopValue returns updated persistent state by value, avoiding a wrapper
// allocation for callers that embed Persistent directly.
func (v Persistent) PopValue() (Persistent, bool) {
	switch v.count {
	case 0:
		return Persistent{}, false
	case 1:
		return Persistent{}, true
	}
	if v.count-v.treeSize() > 1 {
		if v.tailDelta != nil {
			return Persistent{
				count:     v.count - 1,
				height:    v.height,
				root:      v.root,
				tail:      v.tail,
				tailDelta: v.tailDelta.prev,
			}, true
		}
		// Constructor-owned base tails are immutable and can expose a shorter
		// view without copying.
		base := v.baseTail()
		return Persistent{
			count:  v.count - 1,
			height: v.height,
			root:   v.root,
			tail:   newTailBase(base[:len(base)-1]),
		}, true
	}
	newTail := v.sliceFor(v.count - 2)
	newRoot := v.popTail(v.height, v.root)
	newHeight := v.height
	if v.height > 0 && newRoot[1] == nil {
		newRoot = newRoot[0].(node)
		newHeight--
	}
	return Persistent{
		count:  v.count - 1,
		height: newHeight,
		root:   newRoot,
		tail:   newTailBase(newTail),
	}, true
}

// popTail returns a new tree with the last leaf removed.
func (v *Persistent) popTail(level uint, n node) node {
	idx := ((v.count - 2) >> (level * chunkBits)) & chunkMask
	if level > 1 {
		newChild := v.popTail(level-1, n[idx].(node))
		if newChild == nil && idx == 0 {
			return nil
		}
		m := clone(n)
		if newChild == nil {
			// This is needed since `m[idx] = newChild` would store an
			// interface{} with a non-nil type part, which is non-nil
			m[idx] = nil
		} else {
			m[idx] = newChild
		}
		return m
	} else if idx == 0 {
		return nil
	} else {
		m := clone(n)
		m[idx] = nil
		return m
	}
}

func (v *Persistent) SubVector(begin, end int) Vector {
	if begin < 0 || begin > end || end > v.count {
		return nil
	}
	return &subVector{v, begin, end}
}

func (v *Persistent) Iterator() Iterator {
	return newIterator(v)
}

func (v *Persistent) MarshalJSON() ([]byte, error) {
	return marshalJSON(v.Iterator())
}

type subVector struct {
	v     *Persistent
	begin int
	end   int
}

func (s *subVector) Len() int {
	return s.end - s.begin
}

func (s *subVector) Index(i int) (interface{}, bool) {
	if i < 0 || s.begin+i >= s.end {
		return nil, false
	}
	return s.v.Index(s.begin + i)
}

func (s *subVector) Assoc(i int, val interface{}) Vector {
	if i < 0 || s.begin+i > s.end {
		return nil
	} else if s.begin+i == s.end {
		return s.Conj(val)
	}
	return s.v.Assoc(s.begin+i, val).SubVector(s.begin, s.end)
}

func (s *subVector) Conj(val interface{}) Vector {
	return s.v.Assoc(s.end, val).SubVector(s.begin, s.end+1)
}

func (s *subVector) Pop() Vector {
	switch s.Len() {
	case 0:
		return nil
	case 1:
		return Empty
	default:
		return s.v.SubVector(s.begin, s.end-1)
	}
}

func (s *subVector) SubVector(i, j int) Vector {
	return s.v.SubVector(s.begin+i, s.begin+j)
}

func (s *subVector) Iterator() Iterator {
	return newIteratorWithRange(s.v, s.begin, s.end)
}

func (s *subVector) MarshalJSON() ([]byte, error) {
	return marshalJSON(s.Iterator())
}

type iterator struct {
	v        *Persistent
	treeSize int
	index    int
	end      int
	path     []pathEntry
}

type pathEntry struct {
	node  node
	index int
}

func (e pathEntry) current() interface{} {
	return e.node[e.index]
}

func newIterator(v *Persistent) *iterator {
	return newIteratorWithRange(v, 0, v.Len())
}

func newIteratorWithRange(v *Persistent, begin, end int) *iterator {
	it := &iterator{v, v.treeSize(), begin, end, nil}
	if it.index >= it.treeSize {
		return it
	}
	// Find the node for begin, remembering all nodes along the path.
	n := v.root
	for shift := v.height * chunkBits; shift > 0; shift -= chunkBits {
		idx := (begin >> shift) & chunkMask
		it.path = append(it.path, pathEntry{n, idx})
		n = n[idx].(node)
	}
	it.path = append(it.path, pathEntry{n, begin & chunkMask})
	return it
}

func (it *iterator) Elem() interface{} {
	if it.index >= it.treeSize {
		return it.v.tailAt(it.index - it.treeSize)
	}
	return it.path[len(it.path)-1].current()
}

func (it *iterator) HasElem() bool {
	return it.index < it.end
}

func (it *iterator) Next() {
	if it.index+1 >= it.treeSize {
		// Next element is in tail. Just increment the index.
		it.index++
		return
	}
	// Find the deepest level that can be advanced.
	var i int
	for i = len(it.path) - 1; i >= 0; i-- {
		e := it.path[i]
		if e.index+1 < len(e.node) {
			break
		}
	}
	if i == -1 {
		panic("cannot advance; vector iterator bug")
	}
	// Advance on this node, and re-populate all deeper levels.
	it.path[i].index++
	for i++; i < len(it.path); i++ {
		it.path[i] = pathEntry{it.path[i-1].current().(node), 0}
	}
	it.index++
}

type marshalError struct {
	index int
	cause error
}

func (err *marshalError) Error() string {
	return fmt.Sprintf("element %d: %s", err.index, err.cause)
}

func marshalJSON(it Iterator) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	index := 0
	for ; it.HasElem(); it.Next() {
		if index > 0 {
			buf.WriteByte(',')
		}
		elemBytes, err := json.Marshal(it.Elem())
		if err != nil {
			return nil, &marshalError{index, err}
		}
		buf.Write(elemBytes)
		index++
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}
