package throwable

import (
	"errors"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNewWithCause(t *testing.T) {
	cause := errors.New("cause")
	wrapped := New("outer", cause).(error)
	method, ok := lang.FieldOrMethod(wrapped, "getCause")
	if !ok || method.(lang.IFn).Invoke() != cause {
		t.Fatalf("getCause = %v, want %v", method, cause)
	}
}
