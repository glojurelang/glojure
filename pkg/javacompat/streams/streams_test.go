package streams

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStringReaderReadLine(t *testing.T) {
	reader := NewStringReader("first\r\nsecond").(*StringReader)
	if got, want := reader.ReadLine(), "first"; got != want {
		t.Fatalf("first ReadLine = %v, want %v", got, want)
	}
	if got, want := reader.ReadLine(), "second"; got != want {
		t.Fatalf("second ReadLine = %v, want %v", got, want)
	}
	if got := reader.ReadLine(); got != nil {
		t.Fatalf("EOF ReadLine = %v, want nil", got)
	}
}

func TestFileInputStreamReadLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(path, []byte("first\nsecond"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := NewFileInputStream(path).(*FileInputStream)
	defer reader.Close()
	if got, want := reader.ReadLine(), "first"; got != want {
		t.Fatalf("first ReadLine = %v, want %v", got, want)
	}
	if got, want := reader.ReadLine(), "second"; got != want {
		t.Fatalf("second ReadLine = %v, want %v", got, want)
	}
}
