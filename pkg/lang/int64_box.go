package lang

import (
	"sync"
	"sync/atomic"
)

const (
	minCachedInt64 = -128
	maxCachedInt64 = 4096

	// Larger non-negative integers are boxed through blocks that are
	// filled on first use, so loops over positions in a large input do
	// not allocate for every step.
	boxedInt64BlockBits  = 12
	boxedInt64BlockSize  = 1 << boxedInt64BlockBits
	boxedInt64BlockCount = 32
	maxBlockBoxedInt64   = boxedInt64BlockCount*boxedInt64BlockSize - 1
)

var cachedInt64Values = func() []any {
	values := make([]any, maxCachedInt64-minCachedInt64+1)
	for i := range values {
		values[i] = int64(i + minCachedInt64)
	}
	return values
}()

var (
	boxedInt64Blocks   [boxedInt64BlockCount]atomic.Pointer[[boxedInt64BlockSize]any]
	boxedInt64BlocksMu sync.Mutex
)

// BoxInt64 returns a shared interface value for common integers. Go otherwise
// allocates when an int64 escapes through an interface, which is frequent in
// the dynamic numeric path used by both interpreted and generated code.
func BoxInt64(value int64) any {
	if value >= minCachedInt64 && value <= maxCachedInt64 {
		return cachedInt64Values[value-minCachedInt64]
	}
	if value > maxCachedInt64 && value <= maxBlockBoxedInt64 {
		block := boxedInt64Block(int(value >> boxedInt64BlockBits))
		return block[value&(boxedInt64BlockSize-1)]
	}
	return value
}

func boxedInt64Block(index int) *[boxedInt64BlockSize]any {
	if block := boxedInt64Blocks[index].Load(); block != nil {
		return block
	}
	boxedInt64BlocksMu.Lock()
	defer boxedInt64BlocksMu.Unlock()
	if block := boxedInt64Blocks[index].Load(); block != nil {
		return block
	}
	block := new([boxedInt64BlockSize]any)
	base := int64(index) << boxedInt64BlockBits
	for i := range block {
		block[i] = base + int64(i)
	}
	boxedInt64Blocks[index].Store(block)
	return block
}

// Bit flag words made with the bit operators sit far above the block
// range, but a program uses only a few hundred distinct ones and each
// of them many thousand times, so they are boxed once through a small
// direct mapped cache.
const boxedInt64BitsCacheBits = 10

type boxedInt64Entry struct {
	value int64
	boxed any
}

var boxedInt64BitsCache [1 << boxedInt64BitsCacheBits]atomic.Pointer[boxedInt64Entry]

// boxInt64Bits boxes the result of a bit operation, sharing the box for
// values the cache has seen before.
func boxInt64Bits(value int64) any {
	if value >= minCachedInt64 && value <= maxBlockBoxedInt64 {
		return BoxInt64(value)
	}
	hash := uint64(value) * 0x9E3779B97F4A7C15
	slot := &boxedInt64BitsCache[hash>>(64-boxedInt64BitsCacheBits)]
	if entry := slot.Load(); entry != nil && entry.value == value {
		return entry.boxed
	}
	entry := &boxedInt64Entry{value: value, boxed: value}
	slot.Store(entry)
	return entry.boxed
}
