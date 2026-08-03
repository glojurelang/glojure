package lang

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestRecordMapSemantics(t *testing.T) {
	recordType := InternRecordType(
		"record.test",
		"Point",
		"x",
		"y",
	)
	record := NewRecord(recordType, int64(1), int64(2))
	x := NewKeyword("x")
	y := NewKeyword("y")
	color := NewKeyword("color")

	if got := record.ValAt(x); got != int64(1) {
		t.Fatalf("x = %v, want 1", got)
	}
	if got := record.ValAtDefault(color, "missing"); got != "missing" {
		t.Fatalf("missing field = %v, want fallback", got)
	}
	if record.Count() != 2 {
		t.Fatalf("count = %d, want 2", record.Count())
	}

	updated := record.Assoc(x, int64(3)).(RecordValue)
	if updated.RecordType() != recordType || updated.ValAt(x) != int64(3) {
		t.Fatalf("known-field assoc did not retain record type: %v", updated)
	}
	if record.ValAt(x) != int64(1) {
		t.Fatal("known-field assoc mutated the original record")
	}

	extended := updated.Assoc(color, "red").(RecordValue)
	if extended.RecordType() != recordType ||
		extended.ValAt(color) != "red" ||
		extended.Count() != 3 {
		t.Fatalf("extension assoc did not retain record semantics: %v", extended)
	}
	if got := extended.Without(color); got.(RecordValue).RecordType() != recordType {
		t.Fatalf("removing extension key changed type: %T", got)
	}
	withoutBasis := extended.Without(y)
	if _, ok := withoutBasis.(IRecord); ok {
		t.Fatalf("removing basis field retained record type: %T", withoutBasis)
	}
	if withoutBasis.ContainsKey(y) {
		t.Fatal("basis field remains after dissoc")
	}
}

func TestRecordTypeIsResolvableByQualifiedName(t *testing.T) {
	recordType := InternRecordType("record.test", "Resolvable", "value")
	resolved, found := pkgmap.Get("record.test.Resolvable")
	if !found || resolved != recordType {
		t.Fatalf("resolved record type = %v, %v; want %v, true", resolved, found, recordType)
	}
}

func TestRecordMapConstructor(t *testing.T) {
	recordType := InternRecordType(
		"record.test",
		"Person",
		"name",
		"age",
	)
	constructor := NewRecordConstructor(recordType, true)
	record := constructor.Invoke(NewMap(
		NewKeyword("name"), "Ada",
		NewKeyword("extra"), true,
	)).(RecordValue)

	if got := record.ValAt(NewKeyword("name")); got != "Ada" {
		t.Fatalf("name = %v, want Ada", got)
	}
	if got := record.ValAt(NewKeyword("age")); got != nil {
		t.Fatalf("missing basis value = %v, want nil", got)
	}
	if got := record.ValAt(NewKeyword("extra")); got != true {
		t.Fatalf("extension value = %v, want true", got)
	}
	if record.Count() != 3 {
		t.Fatalf("count = %d, want 3", record.Count())
	}
}
