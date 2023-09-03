package lang

type Cons struct {
	meta         IPersistentMap
	hash, hasheq uint32

	first any
	more  ISeq
}

var (
	_ ASeq = (*Cons)(nil)
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
	return c.meta
}

func (c *Cons) WithMeta(meta IPersistentMap) any {
	if meta == c.meta {
		return c
	}

	return &Cons{first: c.first, more: c.more, meta: meta}
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
	return aseqHash(&c.hash, c)
}

func (c *Cons) HashEq() uint32 {
	return aseqHashEq(&c.hasheq, c)
}

func (c *Cons) String() string {
	return aseqString(c)
}
