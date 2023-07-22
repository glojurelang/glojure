package lang

import (
	"testing"
)

func TestIsTruthy(t *testing.T) {
	var v *Var
	if IsTruthy(v) {
		t.Errorf("nil should not be truthy: %v", v)
	}
}
