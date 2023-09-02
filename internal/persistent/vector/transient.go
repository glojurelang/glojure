package vector

// New returns a new Vector with the given elements.
func New(elems ...interface{}) Vector {
	trans := NewTransient(&vector{})
	for _, e := range elems {
		trans.Conj(e)
	}
	return trans.Persistent()
}

type Transient struct {
	count  uint
	height uint
	root   node
	tail   [32]interface{}
}

// NewTransient returns a new transient vector.
func NewTransient(vi Vector) *Transient {
	v := vi.(*vector)
	t := &Transient{
		count:  uint(v.count),
		height: v.height,
		root:   v.root,
	}
	for i := 0; i < len(v.tail); i++ {
		t.tail[i] = v.tail[i]
	}
	return t
}

// Conj adds an element to the end of the vector.
func (t *Transient) Conj(v interface{}) *Transient {
	i := t.count

	// room in tail?
	if i-t.tailoff() < nodeSize {
		t.tail[i&chunkMask] = v
		t.count++
		return t
	}
	// full tail, push into tree
	tailnode := newNode()
	for i := 0; i < len(t.tail); i++ {
		tailnode[i] = t.tail[i]
	}

	t.tail = [32]interface{}{}

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

func (t *Transient) tailoff() uint {
	if t.count < nodeSize {
		return 0
	}
	return ((t.count - 1) >> chunkBits) << chunkBits
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
	return &vector{
		count:  int(t.count),
		height: t.height,
		root:   t.root,
		tail:   t.tail[:t.count-t.tailoff()],
	}
}
