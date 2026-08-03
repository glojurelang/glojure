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

func TestBigDecimalPreservesParsedDecimalText(t *testing.T) {
	input := "23380875855752415049.311059436287553537054164807932529589059912485"
	value, err := NewBigDecimal(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.String(); got != input {
		t.Fatalf("String = %q, want %q", got, input)
	}
}

func TestBigDecimalUnscaledConstructorPreservesDecimalText(t *testing.T) {
	unscaled := newBigIntegerForTest("3145")
	value := newHostBigDecimal(unscaled, int64(3)).(*BigDecimal)
	want := "3.145"
	if got := value.String(); got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
}

func TestBigDecimalFloorAndCeilingScale(t *testing.T) {
	negative, _ := NewBigDecimal("-4.3")
	if got, want := negative.SetScale(0, RoundingModeFloor).ToPlainString(), "-5"; got != want {
		t.Fatalf("floor = %q, want %q", got, want)
	}
	positive, _ := NewBigDecimal("4.3")
	if got, want := positive.SetScale(0, RoundingModeCeiling).ToPlainString(), "5"; got != want {
		t.Fatalf("ceiling = %q, want %q", got, want)
	}
}

func newBigIntegerForTest(value string) *BigInt {
	result, err := NewBigInt(value)
	if err != nil {
		panic(err)
	}
	return result
}
