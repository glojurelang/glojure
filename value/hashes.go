package value

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
)

const (
	keywordHashMask = 0x7334c790
)

func Hash(x interface{}) uint32 {
	if x == nil {
		return 0
	}
	switch x := x.(type) {
	case Object:
		return x.Hash()
	case string:
		h := fnv.New32a()
		h.Write([]byte(x))
		return h.Sum32()
	case int64:
		h := fnv.New32a()
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(x))
		h.Write(b)
		return h.Sum32()
	default:
		panic(fmt.Sprintf("Hash(%T) not implemented", x))
	}
}
