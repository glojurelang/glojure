package lang

// Persistent vectors expose their 32-element leaves as chunks, matching
// Clojure's vector seqs. Bulk consumers can then traverse one leaf at a time.
type vectorSeq struct{ apvSeq }

type vectorChunk struct {
	v          IPersistentVector
	start, end int
}

var (
	_ IChunkedSeq = (*vectorSeq)(nil)
	_ ASeq        = (*vectorSeq)(nil)
	_ IChunk      = (*vectorChunk)(nil)
)

func newVectorSeq(v IPersistentVector, index int) ISeq {
	if index >= v.Count() {
		return nil
	}
	return &vectorSeq{apvSeq: apvSeq{v: v, i: index}}
}

func (s *vectorSeq) Seq() ISeq { return s }

func (s *vectorSeq) Next() ISeq { return newVectorSeq(s.v, s.i+1) }

func (s *vectorSeq) More() ISeq {
	if next := s.Next(); next != nil {
		return next
	}
	return emptyList
}

func (s *vectorSeq) Cons(value any) Conser { return NewCons(value, s) }

func (s *vectorSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	copy := *s
	copy.meta = meta
	return &copy
}

func (s *vectorSeq) chunkEnd() int { return min(s.i+32-s.i%32, s.v.Count()) }

func (s *vectorSeq) ChunkedFirst() IChunk {
	return &vectorChunk{v: s.v, start: s.i, end: s.chunkEnd()}
}

func (s *vectorSeq) ChunkedNext() ISeq { return newVectorSeq(s.v, s.chunkEnd()) }

func (s *vectorSeq) ChunkedMore() ISeq {
	if next := s.ChunkedNext(); next != nil {
		return next
	}
	return emptyList
}

func (c *vectorChunk) Count() int   { return c.end - c.start }
func (c *vectorChunk) xxx_counted() {}
func (c *vectorChunk) Nth(i int) any {
	if i < 0 || i >= c.Count() {
		panic(NewIndexOutOfBoundsError())
	}
	return c.v.Nth(c.start + i)
}

func (c *vectorChunk) NthDefault(i int, fallback any) any {
	if i < 0 || i >= c.Count() {
		return fallback
	}
	return c.v.Nth(c.start + i)
}

func (c *vectorChunk) DropFirst() IChunk {
	if c.Count() == 0 {
		panic(NewIllegalStateError("DropFirst of empty chunk"))
	}
	return &vectorChunk{v: c.v, start: c.start + 1, end: c.end}
}

func (c *vectorChunk) ReduceInit(fn IFn, init any) any {
	result := init
	for i := c.start; i < c.end; i++ {
		result = Apply2(fn, result, c.v.Nth(i))
		if IsReduced(result) {
			return result
		}
	}
	return result
}
