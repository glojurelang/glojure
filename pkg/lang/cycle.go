package lang

import "sync/atomic"

type Cycle struct {
	meta         IPersistentMap
	hash, hasheq uint32

	all     ISeq
	prev    ISeq
	current atomic.Value
	next    atomic.Value
}

var (
	_ ASeq        = (*Cycle)(nil)
	_ IReduce     = (*Cycle)(nil)
	_ IReduceInit = (*Cycle)(nil)
	_ IPending    = (*Cycle)(nil)
)

func newCycle(all, prev, current ISeq) *Cycle {
	c := &Cycle{
		all:  all,
		prev: prev,
	}
	if current != nil {
		c.current.Store(current)
	}
	return c
}

func NewCycle(vals ISeq) ISeq {
	if IsNil(vals) {
		return emptyList
	}
	return newCycle(vals, nil, vals)
}

func (c *Cycle) First() interface{} {
	return c.getCurrent().First()
}

func (c *Cycle) Next() ISeq {
	next := c.next.Load()
	if IsNil(next) {
		next = newCycle(c.all, c.getCurrent(), nil)
		c.next.Store(next)
	}
	return next.(ISeq)
}

func (c *Cycle) getCurrent() ISeq {
	cur := c.current.Load()
	if IsNil(cur) {
		cur = c.prev.Next()
		if IsNil(cur) {
			cur = c.all
		}
		c.current.Store(cur)
	}
	return cur.(ISeq)
}

func (c *Cycle) Cons(o any) Conser {
	return aseqCons(c, o)
}

func (c *Cycle) Count() int {
	return 1 + Count(c.More())
}

func (c *Cycle) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (c *Cycle) Equals(o any) bool {
	return aseqEquals(c, o)
}

func (c *Cycle) Equiv(o any) bool {
	return aseqEquiv(c, o)
}

func (c *Cycle) Hash() uint32 {
	return aseqHash(&c.hash, c)
}

func (c *Cycle) HashEq() uint32 {
	return aseqHashEq(&c.hasheq, c)
}

func (c *Cycle) String() string {
	return aseqString(c)
}

func (c *Cycle) xxx_sequential() {}

func (c *Cycle) More() ISeq {
	sq := c.Next()
	if sq == nil {
		return emptyList
	}
	return sq
}

func (c *Cycle) Seq() ISeq {
	return c
}

func (c *Cycle) Meta() IPersistentMap {
	return c.meta
}

func (c *Cycle) WithMeta(meta IPersistentMap) interface{} {
	if c.meta == meta {
		return c
	}
	cpy := *c
	cpy.meta = meta
	return &cpy
}

////////////////////////////////////////////////////////////////////////////////

func (c *Cycle) IsRealized() bool {
	return !IsNil(c.current.Load())
}

func (c *Cycle) Reduce(f IFn) interface{} {
	s := c.getCurrent()
	ret := s.First()
	for {
		s = s.Next()
		if IsNil(s) {
			s = c.all
		}
		ret = f.Invoke(ret, s.First())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
}

func (c *Cycle) ReduceInit(f IFn, init interface{}) interface{} {
	ret := init
	s := c.getCurrent()
	for {
		ret = f.Invoke(ret, s.First())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
		s = s.Next()
		if IsNil(s) {
			s = c.all
		}
	}
}
