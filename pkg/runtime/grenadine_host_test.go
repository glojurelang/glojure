package runtime

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestGrenadineDigest(t *testing.T) {
	data := []byte("grenadine")
	if got := grenadineDigest(lang.NewKeyword("sha256"), data); got != "f5d1611232b9b6a31561d4f43f839a36fd3cf71ff9fac86e5c6946ae96545090" {
		t.Fatalf("sha256 = %q", got)
	}
}

func TestGrenadineExtractJar(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "library.jar")
	writeTestJar(t, jar, "example/core.clj", "(ns example.core)\n")
	destination := filepath.Join(dir, "extracted")
	if err := grenadineExtractJar(jar, destination); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "example", "core.clj"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "(ns example.core)\n" {
		t.Fatalf("source = %q", got)
	}
	if _, err := os.Stat(filepath.Join(destination, ".grenadine-complete")); err != nil {
		t.Fatal(err)
	}
}

func TestGrenadineExtractJarRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "library.jar")
	writeTestJar(t, jar, "../outside.clj", "bad")
	if err := grenadineExtractJar(jar, filepath.Join(dir, "extracted")); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(dir, "outside.clj")); !os.IsNotExist(err) {
		t.Fatalf("outside path was created: %v", err)
	}
}

func writeTestJar(t *testing.T, path, name, contents string) {
	t.Helper()
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	entry, err := archive.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
