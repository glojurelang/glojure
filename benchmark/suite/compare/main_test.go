package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestManifestAndGeneratedDrivers(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test")
	}
	suitePath := filepath.Join(filepath.Dir(file), "..", "manifest.json")
	suite, suiteDir := readManifest(suitePath)
	if len(suite.Workloads) < 16 {
		t.Fatalf("got %d workloads, want at least 16", len(suite.Workloads))
	}
	driverDir := t.TempDir()
	writeDrivers(suiteDir, driverDir, suite.Workloads)
	for _, value := range suite.Workloads {
		data, err := os.ReadFile(value.driver)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "(println (pr-str (run)))") {
			t.Fatalf("%s has no result driver", value.Name)
		}
	}
}

func TestFormatSamples(t *testing.T) {
	got := formatSamples([]sample{
		{wall: 3 * time.Millisecond, cpu: 5 * time.Millisecond},
		{wall: time.Millisecond, cpu: 2 * time.Millisecond},
		{wall: 2 * time.Millisecond, cpu: 3 * time.Millisecond},
	})
	if got != "2.000 (3.000)" {
		t.Fatalf("formatSamples = %q", got)
	}
}

func TestSelectorSourcesContainEveryWorkload(t *testing.T) {
	values := []workload{
		{
			Name:       "example",
			Namespace:  "benchmark.suite.example",
			Expected:   "42",
			alias:      "example",
			importPath: "benchmark/suite/example",
		},
	}
	for name, source := range map[string]string{
		"glojure": glojureSelectorSource(values),
		"graal":   graalDriverSource(values),
		"let-go":  letGoSelectorSource(values),
	} {
		if !strings.Contains(source, "example") ||
			!strings.Contains(source, "42") {
			t.Errorf("%s selector omitted workload:\n%s", name, source)
		}
	}
}
