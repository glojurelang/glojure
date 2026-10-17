package lang

import "sync/atomic"

type Cons struct {
	attrs atomic.Pointer[seqAttributes]

	first any
	more  ISeq
}

var (
	_ ASeq        = (*Cons)(nil)
	_ IReduce     = (*Cons)(nil)
	_ IReduceInit = (*Cons)(nil)
)

func NewCons(x any, xs any) ISeq {
	switch xs := xs.(type) {
	case nil:
		return NewList(x)
	case ISeq:
		return &Cons{first: x, more: xs}
	default:
		return NewCons(x, Seq(xs))
	}
}

// NewConsSequence copies values into one owned chain of uncounted seq nodes.
// The nodes share an allocation without retaining the caller's mutable slice.
func NewConsSequence(values []any) ISeq {
	if len(values) == 0 {
		return emptyList
	}
	nodes := make([]Cons, len(values))
	for i, value := range values {
		nodes[i].first = value
		if i+1 < len(nodes) {
			nodes[i].more = &nodes[i+1]
		}
	}
	return &nodes[0]
}

func (c *Cons) xxx_sequential() {}

func (c *Cons) Seq() ISeq {
	return c
}

func (c *Cons) First() any {
	return c.first
}

func (c *Cons) Next() ISeq {
	return c.More().Seq()
}

func (c *Cons) More() ISeq {
	if c.more == nil {
		return emptyList
	}
	return c.more
}

func (c *Cons) Meta() IPersistentMap {
	return sequenceMeta(&c.attrs)
}

func (c *Cons) WithMeta(meta IPersistentMap) any {
	if meta == c.Meta() {
		return c
	}

	result := &Cons{first: c.first, more: c.more}
	if meta != nil {
		result.attrs.Store(&seqAttributes{meta: meta})
	}
	return result
}

func (c *Cons) Cons(o any) Conser {
	return aseqCons(c, o)
}

func (c *Cons) Count() int {
	return 1 + Count(c.more)
}

func (c *Cons) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (c *Cons) Equals(o any) bool {
	return aseqEquals(c, o)
}

func (c *Cons) Equiv(o any) bool {
	return aseqEquiv(c, o)
}

func (c *Cons) Hash() uint32 {
	return sequenceHash(&c.attrs, c, false)
}

func (c *Cons) HashEq() uint32 {
	return sequenceHash(&c.attrs, c, true)
}

func (c *Cons) String() string {
	return aseqString(c)
}

func (c *Cons) Reduce(f IFn) any {
	return reduceSeqInit(c.Next(), f, c.First())
}

func (c *Cons) ReduceInit(f IFn, init any) any {
	return reduceSeqInit(c, f, init)
}
