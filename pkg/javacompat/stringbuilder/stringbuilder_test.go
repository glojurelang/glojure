package stringbuilder

import (
	"io"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestStringBuilderSupportsJavaAndWriterCalls(t *testing.T) {
	builder := New("Go").(*StringBuilder)
	if _, ok := any(builder).(io.Writer); !ok {
		t.Fatal("StringBuilder does not implement io.Writer")
	}
	lang.AppendWriter(builder, " + bb!")
	builder.Append("!")
	builder.AppendCodePoint(int('✓'))
	if got, want := builder.ToString(), "Go + bb!!✓"; got != want {
		t.Fatalf("StringBuilder value = %q, want %q", got, want)
	}
}
