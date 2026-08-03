package lang

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestBigDecimalJavaCompatibility(t *testing.T) {
	class, found := pkgmap.Get("java.math.BigDecimal")
	if !found {
		t.Fatal("java.math.BigDecimal was not registered")
	}
	value := NewHostInstance(class, "3.145").(*BigDecimal)
	if got := value.SetScale(2, RoundingModeHalfEven).ToPlainString(); got != "3.14" {
		t.Fatalf("half-even rounding = %q, want 3.14", got)
	}
	if got := value.SetScale(0, RoundingModeDown).ToPlainString(); got != "3" {
		t.Fatalf("down rounding = %q, want 3", got)
	}
	if got := AsBigDecimal(3.0).SetScale(3, RoundingModeHalfEven).ToPlainString(); got != "3.000" {
		t.Fatalf("scaled decimal = %q, want 3.000", got)
	}
}
