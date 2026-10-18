//go:build !glj_aot_runtime

package stdlib

import (
	"io/fs"
	"os"
	"path"
	"reflect"
	"sort"
	"testing"
)

func TestStdLibEmbedsAllSourcesAndOnlySources(t *testing.T) {
	t.Parallel()

	var onDisk []string
	for _, root := range []string{"clojure", "glojure", "clojurestar", "grenadine"} {
		err := fs.WalkDir(os.DirFS("."), root, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && isSource(name) {
				onDisk = append(onDisk, name)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk source directory %q: %v", root, err)
		}
	}

	var embedded []string
	err := fs.WalkDir(StdLib, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !isSource(name) {
			t.Errorf("embedded non-source file %q", name)
		}
		embedded = append(embedded, name)
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded standard library: %v", err)
	}

	sort.Strings(onDisk)
	sort.Strings(embedded)
	if !reflect.DeepEqual(embedded, onDisk) {
		t.Fatalf("embedded sources differ from files on disk:\nembedded: %v\non disk:  %v", embedded, onDisk)
	}
}

func isSource(name string) bool {
	switch path.Ext(name) {
	case ".glj", ".clj", ".cljc":
		return true
	default:
		return false
	}
}
