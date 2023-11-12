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
	newSlice := make([]interface{}, cb.end)
	copy(newSlice, cb.buffer[:cb.end])

	cb.buffer = nil
	cb.end = 0

	return NewSliceChunk(newSlice)
}

func (cb *ChunkBuffer) Count() int {
	return cb.end
}
