package value

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"unsafe"
)

const (
	keywordHashMask = 0x7334c790
	symbolHashMask  = 0x9e3779b9

	// TODO: generic hashes for abitrary go types
	reflectTypeHashMask = 0x49c091a8
)

func Hash(x interface{}) uint32 {
	if IsNil(x) {
		return 0
	}
	if reflect.TypeOf(x).Kind() == reflect.Func {
		// hash of function pointer
		return hashPtr(reflect.ValueOf(x).Pointer())
	}
	switch x := x.(type) {
	case Hasher:
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
	case int:
		return Hash(int64(x))
	case reflect.Type:
		h := getHash()
		h.Write([]byte(x.String()))
		return h.Sum32() ^ reflectTypeHashMask
	default:
		panic(fmt.Sprintf("Hash(%v [%T]) not implemented", x, x))
	}
}

func IdentityHash(x interface{}) uint32 {
	if IsNil(x) {
		return 0
	}
	if reflect.TypeOf(x).Kind() == reflect.Ptr {
		return hashPtr(reflect.ValueOf(x).Pointer())
	}
	return Hash(x)
}

func getHash() hash.Hash32 {
	return fnv.New32a()
}

func hashOrdered(seq ISeq) uint32 {
	h := getHash()
	for ; seq != nil; seq = seq.Next() {
		h.Write(uint32ToBytes(Hash(seq.First())))
	}
	return h.Sum32()
}

func uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, i)
	return b
}

func hashPtr(ptr uintptr) uint32 {
	h := getHash()
	b := make([]byte, unsafe.Sizeof(ptr))
	b[0] = byte(ptr)
	b[1] = byte(ptr >> 8)
	b[2] = byte(ptr >> 16)
	b[3] = byte(ptr >> 24)
	if unsafe.Sizeof(ptr) == 8 {
		b[4] = byte(ptr >> 32)
		b[5] = byte(ptr >> 40)
		b[6] = byte(ptr >> 48)
		b[7] = byte(ptr >> 56)
	}
	h.Write(b)
	return h.Sum32()
}
