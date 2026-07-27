package lang

import (
	"fmt"
	"testing"
)

type transientMapOps interface {
	ITransientMap
	Without(any) ITransientMap
}

type collisionKey int

func (collisionKey) HashEq() uint32 { return 42 }

type controlledHashKey struct {
	id   int
	hash uint32
}

func (key controlledHashKey) HashEq() uint32 { return key.hash }

var transientMapBenchmarkResult any

func newTransientMapForTest(t *testing.T, persistent IPersistentMap) transientMapOps {
	t.Helper()
	editable, ok := persistent.(IEditableCollection)
	if !ok {
		t.Fatalf("%T does not implement IEditableCollection", persistent)
	}
	transient, ok := editable.AsTransient().(transientMapOps)
	if !ok {
		t.Fatalf("%T does not implement transient map operations", editable.AsTransient())
	}
	return transient
}

func assocTransientMap(t *testing.T, transient transientMapOps, key, value any) transientMapOps {
	t.Helper()
	updated, ok := transient.Assoc(key, value).(transientMapOps)
	if !ok {
		t.Fatalf("Assoc returned %T, want transientMapOps", updated)
	}
	return updated
}

func TestTransientMapArrayOperationsAndLifecycle(t *testing.T) {
	original := NewMap(
		NewKeyword("a"), int64(1),
		NewKeyword("b"), int64(2),
	)
	transient := newTransientMapForTest(t, original)

	transient = assocTransientMap(t, transient, NewKeyword("a"), int64(10))
	transient = assocTransientMap(t, transient, nil, "nil-value")
	transient = transient.Without(NewKeyword("b")).(transientMapOps)

	if got := transient.Count(); got != 2 {
		t.Fatalf("Count() = %d, want 2", got)
	}
	if got := transient.ValAt(NewKeyword("a")); got != int64(10) {
		t.Fatalf("updated value = %v, want 10", got)
	}
	if got := transient.ValAt(nil); got != "nil-value" {
		t.Fatalf("nil value = %v, want nil-value", got)
	}
	if got := original.ValAt(NewKeyword("a")); got != int64(1) {
		t.Fatalf("transient mutation changed original value to %v", got)
	}
	if original.ContainsKey(nil) {
		t.Fatal("transient mutation added nil to original map")
	}

	persistent := transient.Persistent().(IPersistentMap)
	if got := persistent.Count(); got != 2 {
		t.Fatalf("persistent count = %d, want 2", got)
	}
	if got := persistent.ValAt(NewKeyword("a")); got != int64(10) {
		t.Fatalf("persistent updated value = %v, want 10", got)
	}
	if persistent.ContainsKey(NewKeyword("b")) {
		t.Fatal("persistent map retained removed key")
	}

	assertTransientMapInvalid(t, "Assoc", func() {
		transient.Assoc("late", true)
	})
	assertTransientMapInvalid(t, "Without", func() {
		transient.Without("missing")
	})
	assertTransientMapInvalid(t, "ValAt", func() {
		transient.ValAt("missing")
	})
	assertTransientMapInvalid(t, "Count", func() {
		transient.Count()
	})
	assertTransientMapInvalid(t, "Persistent", func() {
		transient.Persistent()
	})
}

func TestTransientMapPromotesToHAMTAndHandlesCollisions(t *testing.T) {
	original := NewMap()
	transient := newTransientMapForTest(t, original)

	const entries = 256
	for i := 0; i < entries; i++ {
		transient = assocTransientMap(t, transient, collisionKey(i), i)
	}
	for i := 0; i < entries; i += 3 {
		transient = transient.Without(collisionKey(i)).(transientMapOps)
	}
	for i := 1; i < entries; i += 3 {
		transient = assocTransientMap(t, transient, collisionKey(i), -i)
	}

	persistent := transient.Persistent().(IPersistentMap)
	for i := 0; i < entries; i++ {
		switch i % 3 {
		case 0:
			if persistent.ContainsKey(collisionKey(i)) {
				t.Fatalf("removed collision key %d remains", i)
			}
		case 1:
			if got := persistent.ValAt(collisionKey(i)); got != -i {
				t.Fatalf("updated collision key %d = %v, want %d", i, got, -i)
			}
		case 2:
			if got := persistent.ValAt(collisionKey(i)); got != i {
				t.Fatalf("collision key %d = %v, want %d", i, got, i)
			}
		}
	}
}

func TestTransientMapHAMTCopyOnWrite(t *testing.T) {
	const entries = 512
	key := func(i int) controlledHashKey {
		return controlledHashKey{
			id:   i,
			hash: uint32(i%32) | uint32(i/32)<<5,
		}
	}

	var original Associative = NewMap()
	for i := 0; i < entries; i++ {
		original = original.Assoc(key(i), i)
	}
	originalMap := original.(IPersistentMap)
	transient := newTransientMapForTest(t, originalMap)
	for i := 0; i < entries; i++ {
		switch i % 4 {
		case 0:
			transient = transient.Without(key(i)).(transientMapOps)
		case 1:
			transient = assocTransientMap(t, transient, key(i), -i)
		}
	}
	for i := entries; i < entries+128; i++ {
		transient = assocTransientMap(t, transient, key(i), i)
	}

	firstSnapshot := transient.Persistent().(IPersistentMap)
	if unchanged := firstSnapshot.Assoc(key(2), 2); unchanged != firstSnapshot {
		t.Fatal("same-value assoc on transient-produced persistent map was not a no-op")
	}
	if unchanged := firstSnapshot.Without(key(entries + 1000)); unchanged != firstSnapshot {
		t.Fatal("missing-key removal on transient-produced persistent map was not a no-op")
	}
	for i := 0; i < entries; i++ {
		if got := originalMap.ValAt(key(i)); got != i {
			t.Fatalf("transient mutation changed original key %d to %v", i, got)
		}
		switch i % 4 {
		case 0:
			if firstSnapshot.ContainsKey(key(i)) {
				t.Fatalf("removed key %d remains in first snapshot", i)
			}
		case 1:
			if got := firstSnapshot.ValAt(key(i)); got != -i {
				t.Fatalf("updated key %d = %v, want %d", i, got, -i)
			}
		default:
			if got := firstSnapshot.ValAt(key(i)); got != i {
				t.Fatalf("unchanged key %d = %v, want %d", i, got, i)
			}
		}
	}

	secondTransient := newTransientMapForTest(t, firstSnapshot)
	for i := 1; i < entries; i += 4 {
		secondTransient = assocTransientMap(t, secondTransient, key(i), i*10)
	}
	secondSnapshot := secondTransient.Persistent().(IPersistentMap)
	for i := 1; i < entries; i += 4 {
		if got := firstSnapshot.ValAt(key(i)); got != -i {
			t.Fatalf("second transient changed first snapshot key %d to %v", i, got)
		}
		if got := secondSnapshot.ValAt(key(i)); got != i*10 {
			t.Fatalf("second snapshot key %d = %v, want %d", i, got, i*10)
		}
	}
}

func TestTransientMapMatchesPersistentMapAcrossMixedOperations(t *testing.T) {
	const (
		keys  = 384
		steps = 4096
	)
	key := func(i int) controlledHashKey {
		return controlledHashKey{
			id:   i,
			hash: uint32((i*17)%32) | uint32((i/32)%32)<<5 | uint32(i/1024)<<10,
		}
	}

	transient := newTransientMapForTest(t, NewMap())
	var persistent IPersistentMap = NewMap()
	for step := 0; step < steps; step++ {
		index := (step*73 + step/7) % keys
		if step%5 == 0 || step%11 == 0 {
			transient = transient.Without(key(index)).(transientMapOps)
			persistent = persistent.Without(key(index))
		} else {
			value := step*31 - index
			transient = assocTransientMap(t, transient, key(index), value)
			persistent = persistent.Assoc(key(index), value).(IPersistentMap)
		}
		if got := transient.Count(); got != persistent.Count() {
			t.Fatalf("step %d transient count = %d, persistent count = %d", step, got, persistent.Count())
		}
		if got, want := transient.ValAt(key(index)), persistent.ValAt(key(index)); !Equals(got, want) {
			t.Fatalf("step %d key %d transient value = %v, persistent value = %v", step, index, got, want)
		}
	}

	snapshot := transient.Persistent().(IPersistentMap)
	if !snapshot.Equiv(persistent) {
		t.Fatal("transient result does not equal persistent reference result")
	}
}

func TestTransientMapConjAcceptsMapEntriesAndPairs(t *testing.T) {
	transient := newTransientMapForTest(t, NewMap())
	conjoined, ok := transient.Conj(NewMapEntry("entry", int64(1))).(transientMapOps)
	if !ok {
		t.Fatalf("Conj map entry returned %T", conjoined)
	}
	transient = conjoined
	conjoined, ok = transient.Conj(NewVector("pair", int64(2))).(transientMapOps)
	if !ok {
		t.Fatalf("Conj pair returned %T", conjoined)
	}
	transient = conjoined

	persistent := transient.Persistent().(IPersistentMap)
	if got := persistent.ValAt("entry"); got != int64(1) {
		t.Fatalf("entry value = %v, want 1", got)
	}
	if got := persistent.ValAt("pair"); got != int64(2) {
		t.Fatalf("pair value = %v, want 2", got)
	}
}

func TestTransientMapReducesBuilderAllocations(t *testing.T) {
	const entries = 256
	persistentAllocs := testing.AllocsPerRun(20, func() {
		var result Associative = NewMap()
		for i := 0; i < entries; i++ {
			result = result.Assoc(i, i)
		}
		if result.(Counted).Count() != entries {
			panic("wrong persistent count")
		}
	})
	transientAllocs := testing.AllocsPerRun(20, func() {
		transient := NewMap().(IEditableCollection).AsTransient().(transientMapOps)
		for i := 0; i < entries; i++ {
			transient = transient.Assoc(i, i).(transientMapOps)
		}
		if transient.Persistent().(Counted).Count() != entries {
			panic("wrong transient count")
		}
	})

	if transientAllocs >= persistentAllocs/2 {
		t.Fatalf(
			"transient builder allocated %.0f objects; persistent builder %.0f",
			transientAllocs,
			persistentAllocs,
		)
	}
}

func assertTransientMapInvalid(t *testing.T, operation string, fn func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Errorf("%s after persistent! did not panic", operation)
		}
	}()
	fn()
}

func BenchmarkPersistentMapBuilder(b *testing.B) {
	for _, entries := range []int{8, 32, 256, 1024} {
		b.Run(fmt.Sprintf("%d", entries), func(b *testing.B) {
			for range b.N {
				var result Associative = NewMap()
				for i := 0; i < entries; i++ {
					result = result.Assoc(i, i)
				}
				transientMapBenchmarkResult = result
			}
		})
	}
}

func BenchmarkTransientMapBuilder(b *testing.B) {
	for _, entries := range []int{8, 32, 256, 1024} {
		b.Run(fmt.Sprintf("%d", entries), func(b *testing.B) {
			for range b.N {
				transient := NewMap().(IEditableCollection).AsTransient().(transientMapOps)
				for i := 0; i < entries; i++ {
					transient = transient.Assoc(i, i).(transientMapOps)
				}
				transientMapBenchmarkResult = transient.Persistent()
			}
		})
	}
}
