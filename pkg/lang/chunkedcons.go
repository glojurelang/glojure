package lang

type (
	ChunkedCons struct {
		meta IPersistentMap

		chunk IChunk
		more  ISeq
	}
)

var (
	_ IChunkedSeq = (*ChunkedCons)(nil)
	_ ISeq        = (*ChunkedCons)(nil)
)

func NewChunkedCons(chunk IChunk, more ISeq) *ChunkedCons {
	return &ChunkedCons{
		chunk: chunk,
		more:  more,
	}
}

func (c *ChunkedCons) ChunkedFirst() IChunk {
	return c.chunk
}

func (c *ChunkedCons) ChunkedNext() ISeq {
	return c.ChunkedMore().Seq()
}

func (c *ChunkedCons) ChunkedMore() ISeq {
	if c.more == nil {
		return emptyList
	}
	return c.more
}

func (c *ChunkedCons) First() interface{} {
	return c.chunk.Nth(0)
}

func (c *ChunkedCons) Next() ISeq {
	if c.chunk.Count() > 1 {
		return NewChunkedCons(c.chunk.DropFirst(), c.more)
	}
	return c.ChunkedNext()
}

func (c *ChunkedCons) More() ISeq {
	if c.chunk.Count() > 1 {
		return NewChunkedCons(c.chunk.DropFirst(), c.more)
	}
	if c.more == nil {
		return emptyList
	}
	return c.more
}

func (c *ChunkedCons) Seq() ISeq {
	return c
}

func (c *ChunkedCons) xxx_sequential() {}
