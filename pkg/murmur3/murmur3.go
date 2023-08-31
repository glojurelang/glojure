package murmur3

import "math/bits"

const (
	seed = 0
	c1   = 0xcc9e2d51
	c2   = 0x1b873593
)

type (
	// Seq is a sequence of elements. We duplicate the Seq interface
	// here to avoid a circular dependency.
	Seq interface {
		First() any
		Next() Seq
		More() Seq
	}
)

func HashInt(input int32) uint32 {
	if input == 0 {
		return 0
	}
	k1 := mixK1(uint32(input))
	h1 := mixH1(seed, k1)

	return fmix(h1, 4)
}

func HashLong(input int64) uint32 {
	if input == 0 {
		return 0
	}
	low := uint32(input)
	high := uint32((input >> 32) & 0xffffffff)

	k1 := mixK1(low)
	h1 := mixH1(seed, k1)

	k1 = mixK1(high)
	h1 = mixH1(h1, k1)

	return fmix(h1, 8)
}

func HashOrdered(xs Seq, elHash func(any) uint32) uint32 {
	var n uint32
	var hash uint32 = 1
	for ; xs != nil; xs = xs.Next() {
		eh := elHash(xs.First())
		hash = 31*hash + eh
		n++
	}
	return mixCollHash(hash, n)
}

func HashUnordered(xs Seq, elHash func(any) uint32) uint32 {
	var n uint32
	var hash uint32
	for ; xs != nil; xs = xs.Next() {
		eh := elHash(xs.First())
		hash += eh
		n++
	}
	return mixCollHash(hash, n)
}

func mixCollHash(hash, count uint32) uint32 {
	h1 := uint32(seed)
	k1 := mixK1(hash)
	h1 = mixH1(h1, k1)
	return fmix(h1, count)
}

func mixK1(k1 uint32) uint32 {
	k1 *= c1
	k1 = bits.RotateLeft32(k1, 15)
	k1 *= c2
	return k1
}

func mixH1(h1, k1 uint32) uint32 {
	h1 ^= k1
	h1 = bits.RotateLeft32(h1, 13)
	h1 = h1*5 + 0xe6546b64
	return h1
}

func fmix(h1, length uint32) uint32 {
	h1 ^= length
	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16
	return h1
}
