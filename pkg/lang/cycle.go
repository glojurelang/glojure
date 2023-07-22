//go:generate go run ../../cmd/gen-abstract-class/main.go -class ASeq -struct Cycle -receiver c
package lang

import "sync/atomic"

type Cycle struct {
	meta IPersistentMap

	all     ISeq
	prev    ISeq
	current atomic.Value
	next    atomic.Value
}

var (
	_ ISeq        = (*Cycle)(nil)
	_ IReduce     = (*Cycle)(nil)
	_ IReduceInit = (*Cycle)(nil)
	_ IPending    = (*Cycle)(nil)
	_ Sequential  = (*Cycle)(nil)
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
