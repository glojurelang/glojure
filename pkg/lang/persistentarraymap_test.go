package lang

import (
	"reflect"
	"runtime"
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
	wantKeys := []any{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	if got := mapKeys(m); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("keys = %v, want %v", got, wantKeys)
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

func TestPersistentArrayMapConstructionHandlesTrailingMapEntry(t *testing.T) {
	m := NewPersistentArrayMapAsIfByAssoc([]any{
		"a", 1,
		NewVector("b", 2),
	}).(*Map)

	if got := mapKeys(m); !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("keys = %v, want [a b]", got)
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
	if got := mapKeys(m); !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("keys = %v, want [a b]", got)
	}
}

func TestPersistentArrayMapThresholds(t *testing.T) {
	general := make([]any, 0, hashmapThreshold+2)
	for i := 0; i < hashmapThreshold/2; i++ {
		general = append(general, i, i)
	}
	if _, ok := NewMap(general...).(*Map); !ok {
		t.Fatal("8-entry map did not use an array map")
	}
	general = append(general, hashmapThreshold/2, hashmapThreshold/2)
	if _, ok := NewMap(general...).(*PersistentHashMap); !ok {
		t.Fatal("9-entry general-key map did not use a hash map")
	}

	keywords := make([]any, 0, hashmapThreshold+2)
	for i := 0; i < hashmapThreshold/2; i++ {
		keywords = append(keywords, NewKeyword(string(rune('a'+i))), i)
	}
	if _, ok := NewMap(keywords...).(*Map); !ok {
		t.Fatal("8-entry keyword map did not use an array map")
	}
	keywords = append(keywords, NewKeyword("over-threshold"), 9)
	if _, ok := NewMap(keywords...).(*PersistentHashMap); !ok {
		t.Fatal("9-entry keyword map did not use a hash map")
	}
}

func TestPersistentArrayMapAssocPromotesAtThreshold(t *testing.T) {
	keyVals := make([]any, 0, hashmapThreshold)
	for i := 0; i < hashmapThreshold/2; i++ {
		keyVals = append(keyVals, i, i)
	}
	m := NewMap(keyVals...).(*Map)

	got := m.Assoc("new", 9)
	if _, ok := got.(*PersistentHashMap); !ok {
		t.Fatalf("assoc returned %T, want *PersistentHashMap", got)
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

func TestPersistentArrayMapAssocInvalidatesCachedHashes(t *testing.T) {
	original := NewMap("a", 1).(*Map)
	original.Hash()
	original.HashEq()

	updated := original.Assoc("a", 2).(*Map)
	expected := NewMap("a", 2).(*Map)
	if got, want := updated.Hash(), expected.Hash(); got != want {
		t.Fatalf("hash after assoc = %d, want %d", got, want)
	}
	if got, want := updated.HashEq(), expected.HashEq(); got != want {
		t.Fatalf("hasheq after assoc = %d, want %d", got, want)
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

	if got := mapKeys(original); !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("original keys = %v, want [a b]", got)
	}
	if got := original.ValAt("a"); got != 1 {
		t.Fatalf("original value at a = %v, want 1", got)
	}

	persistent := transient.Persistent().(*Map)
	if got := mapKeys(persistent); !reflect.DeepEqual(got, []any{"a", "c"}) {
		t.Fatalf("persistent keys = %v, want [a c]", got)
	}
	if got := persistent.ValAt("a"); got != 10 {
		t.Fatalf("persistent value at a = %v, want 10", got)
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
