package lang

import "sync/atomic"

// Ordinary sequence nodes carry no metadata or hash cache. Allocate those
// optional attributes only when they are used, publishing caches safely.
type seqAttributes struct {
	meta         IPersistentMap
	hash, hasheq atomic.Uint32
}

func sequenceMeta(attrs *atomic.Pointer[seqAttributes]) IPersistentMap {
	if value := attrs.Load(); value != nil {
		return value.meta
	}
	return nil
}

func sequenceAttributes(attrs *atomic.Pointer[seqAttributes]) *seqAttributes {
	if value := attrs.Load(); value != nil {
		return value
	}
	value := &seqAttributes{}
	if attrs.CompareAndSwap(nil, value) {
		return value
	}
	return attrs.Load()
}

func sequenceHash(attrs *atomic.Pointer[seqAttributes], seq ASeq, equiv bool) uint32 {
	attributes := sequenceAttributes(attrs)
	cache := &attributes.hash
	if equiv {
		cache = &attributes.hasheq
	}
	if hash := cache.Load(); hash != 0 {
		return hash
	}
	var hash uint32
	if equiv {
		aseqHashEq(&hash, seq)
	} else {
		aseqHash(&hash, seq)
	}
	cache.Store(hash)
	return hash
}
