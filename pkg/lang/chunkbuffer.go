package lang

type (
	ChunkBuffer struct {
		buffer []interface{}
		end    int
		addFn  FnFunc1
	}
)

var (
	_ Counted = (*ChunkBuffer)(nil)
)

func NewChunkBuffer(capacity int) *ChunkBuffer {
	cb := &ChunkBuffer{
		buffer: make([]interface{}, capacity),
		end:    0,
	}
	cb.addFn = func(item any) any {
		cb.Add(item)
		return nil
	}
	return cb
}

func (cb *ChunkBuffer) Add(item interface{}) {
	// following Clojure's implementation, we pre-allocate the
	// buffer. additions beyond the initial capacity will cause a
	// runtime error.
	cb.buffer[cb.end] = item
	cb.end++
}

func (cb *ChunkBuffer) Chunk() IChunk {
	newSlice := NewSliceChunk(cb.buffer[:cb.end])

	cb.buffer = nil
	cb.end = 0

	return newSlice
}

func (cb *ChunkBuffer) Count() int {
	return cb.end
}

func (cb *ChunkBuffer) fieldOrMethod(name string) (interface{}, bool) {
	switch name {
	case "add", "Add":
		return cb.addFn, true
	case "chunk", "Chunk":
		return FnFunc0(func() any { return cb.Chunk() }), true
	case "count", "Count":
		return FnFunc0(func() any { return cb.Count() }), true
	}
	return nil, false
}

func (cb *ChunkBuffer) xxx_counted() {}
