package lang

import (
	"runtime"
	"strconv"
	"testing"
)

func TestSmallMapAssocUsesIndependentInlineStorage(t *testing.T) {
	keys := []Keyword{
		NewKeyword("a"),
		NewKeyword("b"),
		NewKeyword("c"),
		NewKeyword("d"),
	}
	original := NewMap(
		keys[0], int64(1),
		keys[1], int64(2),
		keys[2], int64(3),
		keys[3], int64(4),
	).(*Map)

	updated := original.Assoc(keys[1], int64(20)).(*Map)
	if got := original.ValAt(keys[1]); got != int64(2) {
		t.Fatalf("original value = %v, want 2", got)
	}
	if got := updated.ValAt(keys[1]); got != int64(20) {
		t.Fatalf("updated value = %v, want 20", got)
	}

	var result Associative
	key := any(keys[0])
	value := any(int64(10))
	if got := testing.AllocsPerRun(1_000, func() {
		result = original.Assoc(key, value)
	}); got != 1 {
		t.Fatalf("small-map assoc allocated %v objects per call, want 1", got)
	}
	runtime.KeepAlive(result)
}

func TestSmallMapWithMetaKeepsInlineStorageAlive(t *testing.T) {
	key := NewKeyword("key")
	meta := NewMap(NewKeyword("source"), "test").(IPersistentMap)

	withMeta := func() *Map {
		original := NewMap(key, "value").(*Map)
		return original.WithMeta(meta).(*Map)
	}()
	runtime.GC()

	if got := withMeta.ValAt(key); got != "value" {
		t.Fatalf("value after WithMeta and GC = %v, want value", got)
	}
	if got := withMeta.Meta(); got != meta {
		t.Fatalf("Meta() = %v, want %v", got, meta)
	}
}

func TestNewMapUniqueKeysRetainsCompilerOwnedStorage(t *testing.T) {
	keyVals := make([]any, arrayMapHashThreshold)
	for i := 0; i < len(keyVals); i += 2 {
		keyVals[i] = NewKeyword("k" + strconv.Itoa(i/2))
		keyVals[i+1] = int64(i / 2)
	}

	m := NewMapUniqueKeys(keyVals...).(*Map)
	if &m.keyVals[0] != &keyVals[0] {
		t.Fatal("NewMapUniqueKeys copied compiler-owned key/value storage")
	}

	var result IPersistentMap
	if got := testing.AllocsPerRun(1_000, func() {
		result = NewMapUniqueKeys(keyVals...)
	}); got != 1 {
		t.Fatalf("owned-storage map construction allocated %v objects, want 1", got)
	}
	runtime.KeepAlive(result)
}

func TestMapAssocInvalidatesCachedHashes(t *testing.T) {
	key := NewKeyword("key")
	original := NewMap(key, int64(1)).(*Map)
	original.Hash()
	original.HashEq()

	updated := original.Assoc(key, int64(2)).(*Map)
	fresh := NewMap(key, int64(2)).(*Map)

	if got, want := updated.Hash(), fresh.Hash(); got != want {
		t.Fatalf("updated Hash() = %d, want %d", got, want)
	}
	if got, want := updated.HashEq(), fresh.HashEq(); got != want {
		t.Fatalf("updated HashEq() = %d, want %d", got, want)
	}
}

func TestMapAssocIdenticalValueReturnsOriginal(t *testing.T) {
	key := NewKeyword("key")
	value := []int{1, 2, 3}
	original := NewMap(key, value).(*Map)

	if got := original.Assoc(key, value); got != original {
		t.Fatal("Assoc with an identical value did not return the original map")
	}

	equalValue := []int{1, 2, 3}
	updated := original.Assoc(key, equalValue).(*Map)
	if updated == original {
		t.Fatal("Assoc with an equal but non-identical value returned the original map")
	}
	if got := updated.ValAt(key); !Identical(got, equalValue) {
		t.Fatalf("updated value = %v, want the newly associated slice", got)
	}
}

func TestKeywordMapsUseClojureArrayMapThreshold(t *testing.T) {
	keyVals := make([]any, 0, arrayMapKeywordThreshold)
	for i := 0; i < arrayMapKeywordThreshold/2; i++ {
		keyVals = append(keyVals, NewKeyword("k"+strconv.Itoa(i)), int64(i))
	}

	m, ok := NewMap(keyVals...).(*Map)
	if !ok {
		t.Fatalf("keyword map at threshold has type %T, want *Map", NewMap(keyVals...))
	}
	if got := m.Count(); got != arrayMapKeywordThreshold/2 {
		t.Fatalf("Count() = %d, want %d", got, arrayMapKeywordThreshold/2)
	}

	overflow := m.Assoc(NewKeyword("overflow"), int64(1))
	if _, ok := overflow.(*PersistentHashMap); !ok {
		t.Fatalf("keyword map above threshold has type %T, want *PersistentHashMap", overflow)
	}
}

func TestNonKeywordMapUsesGeneralArrayMapThreshold(t *testing.T) {
	keyVals := make([]any, 0, arrayMapHashThreshold+2)
	for i := 0; i < arrayMapHashThreshold/2; i++ {
		keyVals = append(keyVals, i, i)
	}

	m, ok := NewMap(keyVals...).(*Map)
	if !ok {
		t.Fatalf("map at threshold has type %T, want *Map", NewMap(keyVals...))
	}

	overflow := m.Assoc(arrayMapHashThreshold/2, true)
	if _, ok := overflow.(*PersistentHashMap); !ok {
		t.Fatalf("map above threshold has type %T, want *PersistentHashMap", overflow)
	}
}
