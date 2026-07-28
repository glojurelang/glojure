package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestIdentifiers(t *testing.T) {
	if got := goIdentifier("binary-trees"); got != "binary_trees" {
		t.Fatalf("goIdentifier = %q, want binary_trees", got)
	}
	if got := exportedIdentifier("k-nucleotide"); got != "KNucleotide" {
		t.Fatalf("exportedIdentifier = %q, want KNucleotide", got)
	}
}

func TestReadManifest(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	path := filepath.Join(filepath.Dir(file), "..", "manifest.json")
	suite, _ := readManifest(path)
	if len(suite.Workloads) < 6 {
		t.Fatalf("manifest has %d workloads, want at least 6", len(suite.Workloads))
	}
	for _, workload := range suite.Workloads {
		if workload.alias == "" || workload.importPath == "" {
			t.Fatalf("workload was not prepared: %#v", workload)
		}
	}
}
