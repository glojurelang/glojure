package double

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestValueOfAcceptsRatio(t *testing.T) {
	if got, want := ValueOf(lang.NewRatio(3, 2)), 1.5; got != want {
		t.Fatalf("ValueOf ratio = %v, want %v", got, want)
	}
}
