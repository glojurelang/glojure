package value

type Cons struct {
	first interface{}
	rest  ISeq
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
		return &Cons{first: x, rest: xs}
	default:
		return NewCons(x, Seq(xs))
	}
}

func (c *Cons) First() interface{} {
	return c.first
}

func (c *Cons) Next() ISeq {
	return c.rest
}

func (c *Cons) Rest() ISeq {
	return c.rest
}

func (c *Cons) IsEmpty() bool {
	return false
}

// TODO: count

func (c *Cons) Meta() IPersistentMap {
	return c.meta
}

func (c *Cons) WithMeta(meta IPersistentMap) interface{} {
	return &Cons{first: c.first, rest: c.rest, meta: meta}
}
