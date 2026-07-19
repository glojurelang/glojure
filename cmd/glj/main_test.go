//go:build !wasm

package main

import "testing"

func TestDepsGeneratorPackage(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"", depsGenerator},
		{"0.0.0", depsGenerator},
		{"0.6.9-0.20260719092327-0053832ee74f+dirty", depsGenerator},
		{"0.6.9", depsGenerator + "@v0.6.9"},
	}
	for _, test := range tests {
		if got := depsGeneratorPackage(test.version); got != test.want {
			t.Errorf("deps generator for %q = %q, want %q", test.version, got, test.want)
		}
	}
}
