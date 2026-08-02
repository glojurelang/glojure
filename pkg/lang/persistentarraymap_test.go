package lang

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

func mapKeys(m IPersistentMap) []any {
	keys := make([]any, 0, m.Count())
	for seq := m.Seq(); seq != nil; seq = seq.Next() {
		keys = append(keys, seq.First().(IMapEntry).Key())
	}
	return keys
}

func TestPersistentArrayMapConstructionPreservesOrder(t *testing.T) {
	init := []any{
		"a", 1,
		"b", 2,
		"c", 3,
		"d", 4,
		"e", 5,
		"f", 6,
		"g", 7,
		"h", 8,
		"i", 9,
	}
	m, ok := NewPersistentArrayMapAsIfByAssoc(init).(*Map)
	if !ok {
		t.Fatalf("array-map constructor returned %T, want *Map", m)
	}
	if got, want := mapKeys(m), []any{"a", "b", "c", "d", "e", "f", "g", "h", "i"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}

	updated := m.Assoc("a", 10)
	if _, ok := updated.(*Map); !ok {
		t.Fatalf("existing-key assoc returned %T, want *Map", updated)
	}
	grown := m.Assoc("j", 10)
	if _, ok := grown.(*PersistentHashMap); !ok {
		t.Fatalf("new-key assoc returned %T, want *PersistentHashMap", grown)
	}

	init[0] = "changed"
	if !m.ContainsKey("a") || m.ContainsKey("changed") {
		t.Fatal("array-map retained mutable constructor storage")
	}
}

func TestPersistentArrayMapKVReduce(t *testing.T) {
	m := NewMap("a", 1, "b", 2).(*Map)
	result := m.KVReduce(FnFunc(func(args ...any) any {
		return args[0].(int) + args[2].(int)
	}), 0)
	if result != 3 {
		t.Fatalf("KVReduce = %v, want 3", result)
	}
}

func TestPersistentArrayMapConstructionHandlesTrailingMapEntry(t *testing.T) {
	m := NewPersistentArrayMapAsIfByAssoc([]any{
		"a", 1,
		NewVector("b", 2),
	}).(*Map)

	if got, want := mapKeys(m), []any{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	if got := m.ValAt("b"); got != 2 {
		t.Fatalf("value at b = %v, want 2", got)
	}
}

func TestPersistentArrayMapConstructionHandlesDuplicateKeys(t *testing.T) {
	m := NewPersistentArrayMapAsIfByAssoc([]any{
		"a", 1,
		"b", 2,
		"a", 3,
	}).(*Map)

	if got := m.Count(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	if got := m.ValAt("a"); got != 3 {
		t.Fatalf("value at a = %v, want 3", got)
	}
	if got, want := mapKeys(m), []any{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
}

func TestPersistentArrayMapWithoutPreservesTypeOrderAndMeta(t *testing.T) {
	init := []any{
		"a", 1,
		"b", 2,
		"c", 3,
		"d", 4,
		"e", 5,
		"f", 6,
		"g", 7,
		"h", 8,
		"i", 9,
	}
	meta := NewMap(NewKeyword("source"), "test")
	m := NewPersistentArrayMapAsIfByAssoc(init).(*Map).WithMeta(meta).(*Map)

	without := m.Without("e")
	got, ok := without.(*Map)
	if !ok {
		t.Fatalf("without returned %T, want *Map", without)
	}
	if got.Meta() != meta {
		t.Fatal("without discarded metadata")
	}
	wantKeys := []any{"a", "b", "c", "d", "f", "g", "h", "i"}
	if keys := mapKeys(got); !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("keys = %v, want %v", keys, wantKeys)
	}
	if same := m.Without("missing"); same != m {
		t.Fatal("removing a missing key did not return the original map")
	}
}

func TestTransientArrayMapMutatesPrivateStorage(t *testing.T) {
	original := NewMap("a", 1, "b", 2).(*Map)
	transient := original.AsTransient().(*TransientMap)

	key := any("a")
	value := any(10)
	if got := testing.AllocsPerRun(1_000, func() {
		transient.Assoc(key, value)
	}); got != 0 {
		t.Fatalf("transient existing-key assoc allocated %v objects, want 0", got)
	}
	transient.Assoc("c", 3)
	transient.Without("b")

	if got, want := mapKeys(original), []any{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("original keys = %v, want %v", got, want)
	}
	if got := original.ValAt("a"); got != 1 {
		t.Fatalf("original value at a = %v, want 1", got)
	}

	persistent := transient.Persistent().(*Map)
	if got, want := mapKeys(persistent), []any{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("persistent keys = %v, want %v", got, want)
	}
	if got := persistent.ValAt("a"); got != 10 {
		t.Fatalf("persistent value at a = %v, want 10", got)
	}
}

type testStaticKeywordMapStorage struct {
	Map
	values [9]any
}

func newTestStaticKeywordMap(shape *KeywordMapShape, value any) *Map {
	storage := &testStaticKeywordMapStorage{}
	storage.values = [9]any{value, 2, 3, 4, 5, 6, 7, 8, 9}
	return InitStaticKeywordMap(&storage.Map, shape, storage.values[:])
}

func TestStaticKeywordMapStorageCoallocatesMapAndValues(t *testing.T) {
	shape := NewKeywordMapShape("a", "b", "c", "d", "e", "f", "g", "h", "i")
	var result *Map
	if got := testing.AllocsPerRun(1_000, func() {
		result = newTestStaticKeywordMap(shape, 1)
	}); got != 1 {
		t.Fatalf("static keyword map allocated %v objects per call, want 1", got)
	}
	if got := result.ValAt(NewKeyword("i")); got != 9 {
		t.Fatalf("co-allocated map value = %v, want 9", got)
	}
	runtime.KeepAlive(result)

	meta := NewMap(NewKeyword("source"), "test").(IPersistentMap)
	withMeta := func() *Map {
		original := newTestStaticKeywordMap(shape, 1)
		return original.WithMeta(meta).(*Map)
	}()
	runtime.GC()
	if got := withMeta.ValAt(NewKeyword("i")); got != 9 {
		t.Fatalf("co-allocated map value after WithMeta and GC = %v, want 9", got)
	}
}

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

func TestSmallMapWithoutPreservesMetadata(t *testing.T) {
	meta := NewMap(NewKeyword("source"), "test").(IPersistentMap)
	original := NewMap(
		NewKeyword("a"), int64(1),
		NewKeyword("b"), int64(2),
	).(IObj).WithMeta(meta).(*Map)

	removed := original.Without(NewKeyword("a"))
	if got := removed.(IMeta).Meta(); got != meta {
		t.Fatalf("metadata after removal = %v, want %v", got, meta)
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

func TestStaticKeywordMapDeltaCompactionRemainsPersistent(t *testing.T) {
	names := make([]string, keywordMapDeltaMax+2)
	values := make([]any, len(names))
	for i := range names {
		names[i] = "k" + strconv.Itoa(i)
		values[i] = int64(i)
	}
	shape := NewKeywordMapShape(names...)
	base := NewStaticKeywordMap(shape, values...).(*Map)
	updated := base

	for i := 0; i <= keywordMapDeltaMax; i++ {
		updated = updated.Assoc(NewKeyword(names[i]), int64(100+i)).(*Map)
	}
	if updated.keywordDelta != nil {
		t.Fatal("expected the bounded delta overlay to compact")
	}
	for i, name := range names {
		want := int64(i)
		if i <= keywordMapDeltaMax {
			want = int64(100 + i)
		}
		if got := updated.ValAt(NewKeyword(name)); got != want {
			t.Fatalf("updated value for %s = %v, want %v", name, got, want)
		}
		if got := base.ValAt(NewKeyword(name)); got != int64(i) {
			t.Fatalf("base value for %s changed to %v", name, got)
		}
	}

	replaced := base.Assoc(NewKeyword(names[0]), int64(10)).(*Map)
	replacedAgain := replaced.Assoc(NewKeyword(names[0]), int64(20)).(*Map)
	if got := replacedAgain.keywordDelta.depth; got != 1 {
		t.Fatalf("same-slot delta depth = %d, want 1", got)
	}
	if got := replaced.ValAt(NewKeyword(names[0])); got != int64(10) {
		t.Fatalf("replacing a delta changed its predecessor to %v", got)
	}

	key := any(NewKeyword(names[1]))
	value := any(int64(200))
	var result Associative
	if got := testing.AllocsPerRun(1_000, func() {
		result = base.Assoc(key, value)
	}); got != 1 {
		t.Fatalf("shaped-map assoc allocated %v objects per call, want 1", got)
	}
	runtime.KeepAlive(result)

	i := 0
	for seq := updated.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		if entry.Key() != NewKeyword(names[i]) ||
			entry.Val() != updated.ValAt(NewKeyword(names[i])) {
			t.Fatalf("entry %d = %v", i, entry)
		}
		i++
	}
	if i != len(names) {
		t.Fatalf("sequence contained %d entries, want %d", i, len(names))
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
