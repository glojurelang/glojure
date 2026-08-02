package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileListFiles(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "item.txt")
	if err := os.WriteFile(path, []byte("item"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := (&File{Pathname: directory}).ListFiles()
	if len(files) != 1 || files[0].Pathname != path {
		t.Fatalf("ListFiles = %#v, want %q", files, path)
	}
}

func TestCreateTempFile(t *testing.T) {
	file := CreateTempFile("glojure-file-", ".tmp")
	defer os.Remove(file.Pathname)
	if !file.Exists() {
		t.Fatal("temporary file does not exist")
	}
	if got := filepath.Ext(file.Pathname); got != ".tmp" {
		t.Fatalf("extension = %q, want .tmp", got)
	}
}
