package date

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

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
	if !New(int64(42)).(*Date).Equals(New(int64(42))) {
		t.Fatal("dates with equal milliseconds were not equal")
	}
}
