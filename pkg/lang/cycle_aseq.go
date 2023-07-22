// GENERATED CODE. DO NOT EDIT
package lang

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
	if Equal(c.meta, meta) {
		return c
	}
	cpy := *c
	cpy.meta = meta
	return &cpy
}
