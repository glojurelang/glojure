package character

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestToTitleCase(t *testing.T) {
	if got, want := ToTitleCase(lang.Char('f')), lang.Char('F'); got != want {
		t.Fatalf("ToTitleCase = %v, want %v", got, want)
	}
}
