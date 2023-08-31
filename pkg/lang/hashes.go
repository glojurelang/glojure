package lang

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math/big"
	"reflect"
	"unsafe"

	hash2 "bitbucket.org/pcastools/hash"
)

const (
	keywordHashMask = 0x7334c790
	symbolHashMask  = 0x9e3779b9

	// TODO: generic hashes for abitrary go types
	reflectTypeHashMask  = 0x49c091a8
	reflectValueHashMask = 0x49c791a8
)

func HashEq(x any) uint32 {
	return Hash(x)
}

func Hash(x interface{}) uint32 {
	if IsNil(x) {
		return 0
	}

	if IsNumber(x) {
		return hashNumber(x)
	}

	switch x := x.(type) {
	case Hasher:
		return x.Hash()
	case string:
		h := fnv.New32a()
		h.Write([]byte(x))
		return h.Sum32()
	case reflect.Type:
		h := getHash()
		h.Write([]byte(x.String()))
		return h.Sum32() ^ reflectTypeHashMask
	case reflect.Value:
		if !x.IsValid() {
			return reflectValueHashMask
		}
		return Hash(x.Interface()) ^ reflectValueHashMask
	}

	switch reflect.TypeOf(x).Kind() {
	case reflect.Func, reflect.Chan, reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Slice:
		// hash of pointer
		return hashPtr(reflect.ValueOf(x).Pointer())
	}

	panic(fmt.Sprintf("Hash(%v [%T]) not implemented", x, x))
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

func hashNumber(x any) uint32 {
	switch x := x.(type) {
	case int64:
		return hash2.Int64(x)
	case int:
		return hash2.Int64(int64(x))
	case int32:
		return hash2.Int64(int64(x))
	case int16:
		return hash2.Int64(int64(x))
	case int8:
		return hash2.Int64(int64(x))
	case uint64:
		return hash2.Uint64(x)
	case uint:
		return hash2.Uint64(uint64(x))
	case uint32:
		return hash2.Uint64(uint64(x))
	case uint16:
		return hash2.Uint64(uint64(x))
	case uint8:
		return hash2.Uint64(uint64(x))
	case float64:
		if x == 0 {
			return 0
		}
		return hash2.Float64(x)
	case float32:
		if x == 0 {
			return 0
		}
		return hash2.Float32(x)
	case *Ratio:
		return hashNumber(x.Numerator()) ^ hashNumber(x.Denominator())
	case *big.Int:
		if x.IsInt64() {
			return hashNumber(x.Int64())
		}
		return hashNumber(hash2.ByteSlice(x.Bytes()))
	case Hasher:
		return x.Hash()
	}

	panic(fmt.Sprintf("hashNumber(%v [%T]) not implemented", x, x))
}
