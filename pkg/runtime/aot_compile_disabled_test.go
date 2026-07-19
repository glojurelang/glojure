//go:build glj_aot_runtime

package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompactRuntimeRejectsSourceGeneration(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("compileNSToFile did not reject source generation")
		}
		if message := fmt.Sprint(recovered); !strings.Contains(message, "glj_aot_runtime") {
			t.Fatalf("unexpected error: %s", message)
		}
	}()

	compileNSToFile(nil, "example/app")
}
