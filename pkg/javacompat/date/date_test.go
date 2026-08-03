package date

import (
	"bytes"
	"testing"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestDateReadableRepresentation(t *testing.T) {
	value := New(int64(1577934245006)).(*Date)
	var output bytes.Buffer
	value.PrintReadable(&output)
	if got, want := output.String(), `#inst "2020-01-02T03:04:05.006+00:00"`; got != want {
		t.Fatalf("printed = %q, want %q", got, want)
	}
}

func TestDateConstructionAndComparison(t *testing.T) {
	earlier := New(int64(10)).(*Date)
	later := New(int64(11)).(*Date)
	if got := earlier.GetTime(); got != 10 {
		t.Fatalf("GetTime = %d, want 10", got)
	}
	if got := lang.Compare(earlier, later); got >= 0 {
		t.Fatalf("Compare = %d, want negative", got)
	}
}

func TestParseInstantDate(t *testing.T) {
	date := ParseInstantDate("2020-01-02T03:04:05.006Z").(*Date)
	if got, want := date.GetTime(), int64(1577934245006); got != want {
		t.Fatalf("GetTime() = %d, want %d", got, want)
	}
}

func TestDateEquality(t *testing.T) {
	left := New(int64(42)).(*Date)
	right := New(int64(42)).(*Date)
	if !left.Equals(right) {
		t.Fatal("dates with equal milliseconds were not equal")
	}
	if left.HashCode() != right.HashCode() {
		t.Fatal("equal dates did not have equal hash codes")
	}
}

func TestSQLDateCompatibility(t *testing.T) {
	constructed := New(int64(55), int64(6), int64(12)).(*Date)
	valueOf, ok := pkgmap.Get("java.sql.Date.valueOf")
	if !ok {
		t.Fatal("java.sql.Date.valueOf is not registered")
	}
	parsed := lang.Apply(valueOf.(lang.IFn), []any{"1955-07-12"}).(*Date)
	if !constructed.Equals(parsed) {
		t.Fatalf("constructed %v != parsed %v", constructed, parsed)
	}
	instant := time.UnixMilli(parsed.GetTime()).In(time.Local)
	if got, want := instant.Format("2006-01-02"), "1955-07-12"; got != want {
		t.Fatalf("date = %s, want %s", got, want)
	}
}
