package lang

type (
	ChunkedCons struct {
		meta   IPersistentMap
		hash   uint32
		hashEq uint32

		chunk IChunk
		more  ISeq
	}
)

var (
	_ IChunkedSeq = (*ChunkedCons)(nil)
	_ ASeq        = (*ChunkedCons)(nil)
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

func (c *ChunkedCons) First() any {
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

func (c *ChunkedCons) xxx_sequential() {}

func (c *ChunkedCons) Seq() ISeq {
	return c
}

func (c *ChunkedCons) Meta() IPersistentMap {
	return c.meta
}

func (c *ChunkedCons) WithMeta(meta IPersistentMap) any {
	if c.meta == meta {
		return c
	}
	cpy := *c
	cpy.meta = meta
	return &cpy
}

func (c *ChunkedCons) Count() int {
	return aseqCount(c)
}

func (c *ChunkedCons) Cons(o any) Conser {
	return aseqCons(c, o)
}

func (c *ChunkedCons) Empty() IPersistentCollection {
	return asetEmpty()
}

func (c *ChunkedCons) Equiv(o any) bool {
	return aseqEquiv(c, o)
}

func (c *ChunkedCons) Equals(o any) bool {
	return aseqEquals(c, o)
}

func (c *ChunkedCons) Hash() uint32 {
	return aseqHash(&c.hash, c)
}

func (c *ChunkedCons) HashEq() uint32 {
	return aseqHashEq(&c.hashEq, c)
}

func (c *ChunkedCons) String() string {
	return aseqString(c)
}
