package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestOwnedVectorPreservesNestedPersistentVersionsAndMetadata(t *testing.T) {
	innerMeta := lang.NewMap(lang.NewKeyword("inner"), true).(lang.IPersistentMap)
	outerMeta := lang.NewMap(lang.NewKeyword("outer"), true).(lang.IPersistentMap)
	inner := lang.NewVector(int64(1), int64(2)).
		WithMeta(innerMeta).(*lang.Vector)
	original := lang.NewVector(inner).
		WithMeta(outerMeta).(*lang.Vector)

	owned, ok := NewOwnedVector(original, 2)
	if !ok {
		t.Fatal("nested vector shape was rejected")
	}
	updated := owned.AssocIn2Copy(0, 1, int64(20))
	result := updated.Persistent()

	if got := original.Nth(0).(*lang.Vector).Nth(1); got != int64(2) {
		t.Fatalf("original nested value = %v, want 2", got)
	}
	if got := result.Nth(0).(*lang.Vector).Nth(1); got != int64(20) {
		t.Fatalf("updated nested value = %v, want 20", got)
	}
	if result.Meta() != outerMeta ||
		result.Nth(0).(*lang.Vector).Meta() != innerMeta {
		t.Fatal("persistent boundary did not preserve vector metadata")
	}
}

func TestOwnedVectorRejectsMismatchedShapeBeforeMutation(t *testing.T) {
	if _, ok := NewOwnedVector(lang.NewVector(int64(1)), 2); ok {
		t.Fatal("flat vector accepted as a nested vector")
	}
	if _, ok := NewOwnedVector(lang.NewList(int64(1)), 1); ok {
		t.Fatal("list accepted as an owned vector")
	}
}

func TestOwnedVectorSnapshotsPreservePreMutationReads(t *testing.T) {
	original := lang.NewVector(lang.NewVector(int64(1), int64(2)))
	owned, ok := NewOwnedVector(original, 2)
	if !ok {
		t.Fatal("nested vector shape was rejected")
	}
	snapshot := owned.NestedSnapshot(0)
	updated := owned.AssocIn2Copy(0, 0, int64(10))
	if got := snapshot.Nth(0); got != int64(1) {
		t.Fatalf("snapshot read = %v, want original value 1", got)
	}
	if got := updated.Nested(0).Nth(0); got != int64(10) {
		t.Fatalf("owned read = %v, want updated value 10", got)
	}
}

func TestOwnedVectorAssocRejectsUnsupportedIndices(t *testing.T) {
	owned, ok := NewOwnedVector(lang.NewVector(int64(1)), 1)
	if !ok {
		t.Fatal("vector shape was rejected")
	}
	assertOwnedVectorPanic(t, func() { owned.Assoc(-1, int64(0)) })
	assertOwnedVectorPanic(t, func() { owned.Assoc(2, int64(0)) })
	owned.Assoc(1, int64(2))
	if got := owned.Persistent().Count(); got != 2 {
		t.Fatalf("appended vector count = %d, want 2", got)
	}
}

func TestOwnedVectorNestedAssocCanAppendAssociativeValue(t *testing.T) {
	owned, ok := NewOwnedVector(
		lang.NewVector(lang.NewVector(int64(1))),
		2,
	)
	if !ok {
		t.Fatal("nested vector shape was rejected")
	}
	key := lang.NewKeyword("key")
	updated := owned.AssocIn2Copy(1, key, int64(2))
	appended := updated.Persistent().Nth(1)
	if got := lang.Get(appended, key); got != int64(2) {
		t.Fatalf("appended associative value = %v, want 2", got)
	}
}

func TestOwnedVectorPersistentReusesUnchangedNestedVectors(t *testing.T) {
	left := lang.NewVector(int64(1))
	right := lang.NewVector(int64(2))
	original := lang.NewVector(left, right)
	owned, ok := NewOwnedVector(original, 2)
	if !ok {
		t.Fatal("nested vector shape was rejected")
	}
	updated := owned.AssocIn2Copy(0, 0, int64(10))
	result := updated.Persistent()
	if result == original {
		t.Fatal("updated outer vector reused its original identity")
	}
	if result.Nth(0) == left {
		t.Fatal("updated nested vector reused its original identity")
	}
	if result.Nth(1) != right {
		t.Fatal("unchanged nested vector did not retain its identity")
	}
}

func TestOwnedVectorAssocPathCopiesArbitraryDepth(t *testing.T) {
	original := lang.NewVector(
		lang.NewVector(lang.NewVector(int64(1), int64(2))),
	)
	owned, ok := NewOwnedVector(original, 3)
	if !ok {
		t.Fatal("three-level vector was rejected")
	}
	updated := owned.AssocPathCopy([]int{0, 0, 1}, int64(9)).Persistent()
	want := lang.NewVector(
		lang.NewVector(lang.NewVector(int64(1), int64(9))),
	)
	if !lang.Equals(updated, want) {
		t.Fatalf("updated value = %v, want %v", updated, want)
	}
	if !lang.Equals(
		original,
		lang.NewVector(lang.NewVector(lang.NewVector(int64(1), int64(2)))),
	) {
		t.Fatalf("original value was mutated: %v", original)
	}
}

func TestOwnedVectorNestedSnapshotCanCopyDeeperPath(t *testing.T) {
	original := lang.NewVector(
		lang.NewVector(lang.NewVector(int64(1), int64(2))),
	)
	owned, ok := NewOwnedVector(original, 3)
	if !ok {
		t.Fatal("three-level vector was rejected")
	}
	snapshot := owned.NestedSnapshot(0)
	updated := snapshot.AssocPath([]int{0, 1}, int64(9)).Persistent()
	want := lang.NewVector(lang.NewVector(int64(1), int64(9)))
	if !lang.Equals(updated, want) {
		t.Fatalf("snapshot update = %v, want %v", updated, want)
	}
	if !lang.Equals(
		original,
		lang.NewVector(lang.NewVector(lang.NewVector(int64(1), int64(2)))),
	) {
		t.Fatalf("snapshot update mutated original: %v", original)
	}
}

func assertOwnedVectorPanic(t *testing.T, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected vector operation to panic")
		}
	}()
	call()
}
