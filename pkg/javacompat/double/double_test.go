package double

import (
	"math"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestValueOfAcceptsRatio(t *testing.T) {
	if got, want := ValueOf(lang.NewRatio(3, 2)), 1.5; got != want {
		t.Fatalf("ValueOf ratio = %v, want %v", got, want)
	}
}

func TestValueOfReturnsInfinityOnOverflow(t *testing.T) {
	if got := ValueOf("1e9999"); !math.IsInf(got, 1) {
		t.Fatalf("ValueOf overflow = %v, want +Inf", got)
	}
	if got := ValueOf("-1e9999"); !math.IsInf(got, -1) {
		t.Fatalf("ValueOf negative overflow = %v, want -Inf", got)
	}
}
