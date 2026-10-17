package lang

import (
	"math/rand"
	"testing"
)

type buildCollisionKey struct {
	id   int
	hash uint32
}

func (k buildCollisionKey) HashEq() uint32 { return k.hash }
func (k buildCollisionKey) Equiv(other any) bool {
	value, ok := other.(buildCollisionKey)
	return ok && k.id == value.id
}

func TestBulkHashMapMatchesPersistentInsertion(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for _, size := range []int{0, 1, 7, 8, 16, 17, 32, 33, 128, 1024} {
		pairs := make([]any, 0, 2*size)
		var want IPersistentMap = emptyPersistentHashMap
		for i := 0; i < size; i++ {
			var key any
			switch i % 7 {
			case 0:
				key = nil
			case 1:
				key = int64(random.Intn(20))
			case 2:
				key = NewSymbol("symbol" + string(rune('a'+random.Intn(26))))
			case 3:
				key = NewVector(int64(random.Intn(20)))
			case 4:
				key = buildCollisionKey{id: random.Intn(20), hash: 31}
			case 5:
				key = buildCollisionKey{id: 100 + i, hash: uint32(i)<<25 | 31}
			default:
				key = NewKeyword("key" + string(rune('a'+random.Intn(26))))
			}
			value := any(i)
			if i%11 == 0 {
				value = nil
			}
			pairs = append(pairs, key, value)
			want = want.Assoc(key, value).(IPersistentMap)
		}
		got := NewPersistentHashMap(pairs...)
		if got.Count() != want.Count() || !Equals(got, want) || !Equals(want, got) {
			t.Fatalf("bulk map differs at size %d: got %v, want %v", size, got, want)
		}
		// Mutating constructor input cannot affect the published map. Later
		// persistent updates and removals cannot mutate the original either.
		for i := range pairs {
			pairs[i] = nil
		}
		snapshot := got
		for seq := want.Seq(); seq != nil; seq = seq.Next() {
			entry := seq.First().(IMapEntry)
			got = got.Assoc(entry.Key(), -1).(IPersistentMap)
			if !Equals(snapshot, want) {
				t.Fatal("assoc changed a bulk snapshot")
			}
			got = got.Without(entry.Key())
			if got.ContainsKey(entry.Key()) {
				t.Fatal("without retained removed key")
			}
			if !Equals(snapshot, want) {
				t.Fatal("without changed a bulk snapshot")
			}
		}
	}
}

func TestBulkHashMapPreservesFirstKeyAndLastValue(t *testing.T) {
	first, last := NewSymbol("same"), NewSymbol("same")
	pairs := []any{first, 0, "a", 1, "b", 2, "c", 3,
		"d", 4, "e", 5, "f", 6, last, 7}
	result := NewPersistentHashMap(pairs...)
	entry := result.EntryAt(last)
	if entry.Key() != first || entry.Val() != 7 || result.Count() != 7 {
		t.Fatalf("duplicate handling = %v", result)
	}
}
