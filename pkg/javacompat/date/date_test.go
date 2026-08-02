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
