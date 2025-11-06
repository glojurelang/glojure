package runtime_test

import (
	"os"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/runtime"
)

func TestGetLoadPath(t *testing.T) {
	// Test that GetLoadPath returns a valid slice of strings
	paths := runtime.GetLoadPath()
	if len(paths) == 0 {
		t.Fatal("GetLoadPath returned empty slice")
	}

	// Should contain at least the current directory and stdlib
	foundCurrent := false
	foundStdlib := false
	for _, path := range paths {
		if path == "." {
			foundCurrent = true
		}
		if path == "<StdLib>" || strings.Contains(path, "stdlib") {
			foundStdlib = true
		}
	}

	if !foundCurrent {
		t.Error("Load path should contain current directory '.'")
	}
	if !foundStdlib {
		t.Error("Load path should contain stdlib")
	}
}

func TestGLJPATHEnvironmentVariable(t *testing.T) {
	// This test would need to be run in a subprocess to avoid affecting
	// the global loadPath, but for now we'll just verify the structure
	paths := runtime.GetLoadPath()

	// Verify all paths are strings
	for i, path := range paths {
		if path == "" {
			t.Errorf("Path at index %d is empty", i)
		}
	}

	// The expected order is:
	// [...GLJPATH dirs, current dir, <StdLib>]
	if len(paths) > 0 {
		// Last element should always be <StdLib>
		last := paths[len(paths)-1]
		if last != "<StdLib>" {
			t.Errorf("Last path should be '<StdLib>', got: %s", last)
		}

		// If no GLJPATH, order should be [current dir, <StdLib>]
		if os.Getenv("GLJPATH") == "" {
			if len(paths) >= 2 {
				secondToLast := paths[len(paths)-2]
				if secondToLast != "." {
					t.Errorf("Second to last path should be '.', got: %s", secondToLast)
				}
			}
		}
	}
}