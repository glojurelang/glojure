package lang

type (
	ChunkBuffer struct {
		buffer []interface{}
		end    int
	}
)

var (
	_ Counted = (*ChunkBuffer)(nil)
)

func NewChunkBuffer(capacity int) *ChunkBuffer {
	return &ChunkBuffer{
		buffer: make([]interface{}, capacity),
		end:    0,
	}
}

func (cb *ChunkBuffer) Add(item interface{}) {
	// following Clojure's implementation, we pre-allocate the
	// buffer. additions beyond the initial capacity will cause a
	// runtime error.
	cb.buffer[cb.end] = item
	cb.end++
}

func (cb *ChunkBuffer) Chunk() IChunk {
	ret := NewSliceChunk(cb.buffer)
	cb.buffer = nil
	return ret
}

func (cb *ChunkBuffer) Count() int {
	return cb.end
}
