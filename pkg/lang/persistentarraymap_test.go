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

func TestStaticKeywordMapBehavesLikePersistentMap(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	shape := NewKeywordMapShape(names...)
	values := []any{
		int64(1), int64(2), int64(3),
		int64(4), int64(5), int64(6),
		int64(7), int64(8), int64(9),
	}
	shaped := NewStaticKeywordMap(shape, values...).(*Map)

	if got := shaped.Count(); got != len(values) {
		t.Fatalf("Count() = %d, want %d", got, len(values))
	}
	for i, name := range names {
		if got := shaped.ValAt(NewKeyword(name)); got != values[i] {
			t.Fatalf("value for %s = %v, want %v", name, got, values[i])
		}
	}
	if got := shaped.ValAt("a"); got != nil {
		t.Fatalf("non-keyword lookup = %v, want nil", got)
	}

	updated := shaped.Assoc(NewKeyword("e"), int64(50)).(*Map)
	if got := updated.ValAt(NewKeyword("e")); got != int64(50) {
		t.Fatalf("updated value = %v, want 50", got)
	}
	if got := shaped.ValAt(NewKeyword("e")); got != int64(5) {
		t.Fatalf("original value changed to %v", got)
	}

	expanded := shaped.Assoc(NewKeyword("j"), int64(10)).(IPersistentMap)
	if expanded.Count() != 10 || expanded.ValAt(NewKeyword("j")) != int64(10) {
		t.Fatalf("expanded map = %v", expanded)
	}
	removed := shaped.Without(NewKeyword("e"))
	if removed.Count() != 8 || removed.ContainsKey(NewKeyword("e")) {
		t.Fatalf("map after remove = %v", removed)
	}

	var ordinaryArgs []any
	for i, name := range names {
		ordinaryArgs = append(ordinaryArgs, NewKeyword(name), values[i])
	}
	ordinary := NewMap(ordinaryArgs...)
	if !Equals(shaped, ordinary) || !Equals(ordinary, shaped) {
		t.Fatalf("shaped and ordinary maps differ: %v != %v", shaped, ordinary)
	}
	if shaped.Hash() != ordinary.(Hasher).Hash() ||
		shaped.HashEq() != ordinary.(IHashEq).HashEq() {
		t.Fatalf("shaped and ordinary map hashes differ")
	}

	meta := NewMap(NewKeyword("source"), "test").(IPersistentMap)
	withMeta := shaped.WithMeta(meta).(*Map)
	if withMeta.Meta() != meta {
		t.Fatalf("metadata was not retained")
	}
	if got := withMeta.Assoc(NewKeyword("j"), int64(10)).(IMeta).Meta(); got != meta {
		t.Fatalf("metadata after expansion = %v, want %v", got, meta)
	}
	if got := withMeta.Without(NewKeyword("e")).(IMeta).Meta(); got != meta {
		t.Fatalf("metadata after removal = %v, want %v", got, meta)
	}

	i := 0
	for seq := shaped.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		if entry.Key() != NewKeyword(names[i]) || entry.Val() != values[i] {
			t.Fatalf("entry %d = %v", i, entry)
		}
		i++
	}
	if i != len(values) {
		t.Fatalf("sequence contained %d entries, want %d", i, len(values))
	}
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
