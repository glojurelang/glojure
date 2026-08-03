package processbuilder

import (
	"io"
	"testing"
)

func TestProcessBuilderRunsCommand(t *testing.T) {
	builder := New([]string{"sh", "-c", "printf %s \"$GOBB_PROCESS_TEST\""})
	builder.Environment().Put("GOBB_PROCESS_TEST", "ok")
	process := builder.Start()
	if got := process.WaitFor(); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}
	output, err := io.ReadAll(process.GetInputStream())
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "ok" {
		t.Fatalf("output = %q, want ok", output)
	}
}
