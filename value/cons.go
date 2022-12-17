package value

type Cons struct {
	first interface{}
	more  ISeq
	meta  IPersistentMap
}

var (
	_ ISeq = (*Cons)(nil)
)

func NewCons(x interface{}, xs interface{}) ISeq {
	switch xs := xs.(type) {
	case nil:
		return NewList([]interface{}{x})
	case ISeq:
		return &Cons{first: x, more: xs}
	default:
		return NewCons(x, Seq(xs))
	}
}

func (c *Cons) Seq() ISeq {
	return c
}

func (c *Cons) First() interface{} {
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

// TODO: count

func (c *Cons) Meta() IPersistentMap {
	return c.meta
}

func (c *Cons) WithMeta(meta IPersistentMap) interface{} {
	return &Cons{first: c.first, more: c.more, meta: meta}
}
