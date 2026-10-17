package lang

import (
	"fmt"
	"sync"
	"testing"
)

func TestArrayMapHashAndEquivalenceAcrossLayouts(t *testing.T) {
	shape := NewKeywordMapShape("a", "b")
	for _, m := range []*Map{
		NewMap().(*Map),
		NewMap("a", nil, "b", int64(2)).(*Map),
		NewStaticKeywordMap(shape, nil, int64(2)).(*Map),
		NewStaticKeywordMap(shape, nil, int64(2)).Assoc(NewKeyword("b"), int64(3)).(*Map),
	} {
		var hash, hasheq uint32
		wantHash := apersistentmapHash(&hash, m)
		wantEq := apersistentmapHashEq(&hasheq, m)
		var workers sync.WaitGroup
		for i := 0; i < 16; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				for n := 0; n < 10; n++ {
					if m.Hash() != wantHash || m.HashEq() != wantEq {
						t.Error("direct storage hashing changed the map hash")
					}
					tagged := m.WithMeta(NewMap("tag", true)).(*Map)
					if tagged.HashEq() != wantEq {
						t.Error("metadata changed the hash")
					}
				}
			}()
		}
		workers.Wait()
	}
}

func TestLargeArrayMapDeduplicationPreservesOrderAndKeys(t *testing.T) {
	if HashEq(true) != HashEq(NewChar(1231)) {
		t.Fatal("collision fixture changed")
	}
	for _, extension := range []bool{false, true} {
		pairs := []any{true, "boolean", NewChar(1231), "character", nil, "nil"}
		for i := 0; i < 70; i++ {
			key := NewMap(NewKeyword("Str"), fmt.Sprintf("key-%d", i%20))
			pairs = append(pairs, key, int64(i))
		}
		if extension {
			// An equality extension without a corresponding hash must retain
			// comparison-only duplicate handling, including the first key object.
			pairs = append(pairs, keywordEqualValue{keyword: NewKeyword("extension")}, 1,
				NewKeyword("extension"), 2)
		}
		want := newPersistentArrayMapAsIfByAssocComplexPath(pairs, false)
		got := NewPersistentArrayMapAsIfByAssoc(pairs)
		if !Equals(got, want) || !Equals(NewVector(mapKeys(got)...), NewVector(mapKeys(want)...)) {
			t.Fatalf("deduplication changed value or key order: %v", got)
		}
		for i := range pairs {
			pairs[i] = nil
		}
		if !Equals(got, want) {
			t.Fatal("deduplication retained mutable input")
		}
	}
}
