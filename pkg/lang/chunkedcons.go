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
	_ IReduce     = (*ChunkedCons)(nil)
	_ IReduceInit = (*ChunkedCons)(nil)
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
	return Count(c)
}

func (c *ChunkedCons) Reduce(fn IFn) any {
	initial := c.First()
	if next := c.Next(); next != nil {
		return reduceSeqInit(next, fn, initial)
	}
	return initial
}

func (c *ChunkedCons) ReduceInit(fn IFn, init any) any {
	return reduceSeqInit(c, fn, init)
}

func reduceSeqInit(seq ISeq, fn IFn, init any) any {
	result := init
	for seq != nil {
		if chunked, ok := seq.(IChunkedSeq); ok {
			result = chunked.ChunkedFirst().ReduceInit(fn, result)
			if IsReduced(result) {
				return result.(IDeref).Deref()
			}
			seq = chunked.ChunkedNext()
		} else {
			result = Apply2(fn, result, seq.First())
			if IsReduced(result) {
				return result.(IDeref).Deref()
			}
			seq = seq.Next()
		}
	}
	return result
}

func (c *ChunkedCons) Cons(o any) Conser {
	return aseqCons(c, o)
}

func (c *ChunkedCons) Empty() IPersistentCollection {
	return aseqEmpty()
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
