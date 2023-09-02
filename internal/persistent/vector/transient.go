package vector

import (
	"errors"
	"sync/atomic"
)

// New returns a new Vector with the given elements.
func New(elems ...interface{}) Vector {
	trans := NewTransient(&vector{})
	for _, e := range elems {
		trans.Conj(e)
	}
	return trans.Persistent()
}

type Transient struct {
	count  int
	height uint
	root   node
	tail   []interface{}

	persistent atomic.Bool
}

// NewTransient returns a new transient vector.
func NewTransient(vi Vector) *Transient {
	v := vi.(*vector)
	t := &Transient{
		count:  v.count,
		height: v.height,
		root:   v.root,
		tail:   make([]interface{}, nodeSize),
	}
	for i := 0; i < len(v.tail); i++ {
		t.tail[i] = v.tail[i]
	}
	return t
}

func (t *Transient) ensureEditable() {
	if t.persistent.Load() {
		panic(errors.New("transient used after persistent! call"))
	}
}

func (t *Transient) Count() int {
	return t.count
}

func (t *Transient) Index(i int) (any, bool) {
	if i < 0 || i >= t.count {
		return nil, false
	}

	// The following is very similar to sliceFor, but is implemented separately
	// to avoid unnecessary copying.
	if i >= t.treeSize() {
		return t.tail[i&chunkMask], true
	}
	n := t.root
	for shift := t.height * chunkBits; shift > 0; shift -= chunkBits {
		n = n[(i>>shift)&chunkMask].(node)
	}
	return n[i&chunkMask], true
}

// Conj adds an element to the end of the vector.
func (t *Transient) Conj(v interface{}) *Transient {
	t.ensureEditable()
	i := t.count

	// room in tail?
	if uint(i)-t.tailoff() < nodeSize {
		t.tail[i&chunkMask] = v
		t.count++
		return t
	}
	// full tail, push into tree
	tailnode := newNode()
	for i := 0; i < len(t.tail); i++ {
		tailnode[i] = t.tail[i]
	}

	t.tail = make([]interface{}, nodeSize)

	t.tail[0] = v
	newheight := t.height

	var newroot node

	//overflow root?
	if (t.count >> chunkBits) > (1 << (t.height * chunkBits)) {
		newroot = newNode()
		newroot[0] = t.root
		newroot[1] = newPath(t.height, tailnode)
		newheight++
	} else {
		newroot = t.pushTail(t.height, t.root, tailnode)
	}

	t.root = newroot
	t.height = newheight
	t.count++
	return t
}

func (t *Transient) Assoc(i int, val interface{}) *Transient {
	t.ensureEditable()
	if i < 0 || i > t.count {
		return nil
	} else if i == t.count {
		return t.Conj(val)
	}
	if i >= t.treeSize() {
		t.tail[i&chunkMask] = val
		return t
	}
	n := t.root
	for shift := t.height * chunkBits; shift > 0; shift -= chunkBits {
		n = n[(i>>shift)&chunkMask].(node)
	}
	n[i&chunkMask] = val
	return t
}

func (t *Transient) tailoff() uint {
	if t.count < nodeSize {
		return 0
	}
	return uint(((t.count - 1) >> chunkBits) << chunkBits)
}

func (t *Transient) pushTail(height uint, n, tail node) node {
	if height == 0 {
		return tail
	}

	idx := ((t.count - 1) >> (height * chunkBits)) & chunkMask
	m := clone(n)
	child := n[idx]
	if child == nil {
		m[idx] = newPath(height-1, tail)
	} else {
		m[idx] = t.pushTail(height-1, child.(node), tail)
	}
	return m
}

// persistent returns a persistent vector from the transient vector.
func (t *Transient) Persistent() *vector {
	t.persistent.Store(true)
	return &vector{
		count:  int(t.count),
		height: t.height,
		root:   t.root,
		tail:   t.tail[:uint(t.count)-t.tailoff()],
	}
}

func (t *Transient) Pop() *Transient {
	t.ensureEditable()

	if t.count == 0 {
		return t
	}
	if t.count == 1 {
		t.count = 0
		return t
	}

	if t.count-t.treeSize() > 1 {
		t.count--
		return t
	}
	newTail := t.sliceFor(t.count - 2)
	newRoot := t.popTail(t.height, t.root) // TODO: more efficient transient popTail
	newHeight := t.height
	if t.height > 0 && newRoot[1] == nil {
		newRoot = newRoot[0].(node)
		newHeight--
	}
	t.root = newRoot
	t.height = newHeight
	t.count--
	t.tail = newTail
	return t
}

func (t *Transient) popTail(level uint, n node) node {
	idx := ((t.count - 2) >> (level * chunkBits)) & chunkMask
	if level > 1 {
		newChild := t.popTail(level-1, n[idx].(node))
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

func (t *Transient) treeSize() int {
	if t.count < tailMaxLen {
		return 0
	}
	return ((t.count - 1) >> chunkBits) << chunkBits
}

func (t *Transient) sliceFor(i int) []interface{} {
	if i >= t.treeSize() {
		return t.tail
	}
	n := t.root
	for shift := t.height * chunkBits; shift > 0; shift -= chunkBits {
		n = n[(i>>shift)&chunkMask].(node)
	}
	return n[:]
}
