package math

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestFloatFunctionsAcceptRatios(t *testing.T) {
	floor := fn1Float64(func(x float64) float64 { return x })
	if got, want := lang.Apply1(floor, lang.NewRatio(3, 2)), 1.5; got != want {
		t.Fatalf("ratio conversion = %v, want %v", got, want)
	}
}
