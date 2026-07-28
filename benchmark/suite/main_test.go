package main

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseRuntime(t *testing.T) {
	got, err := parseRuntime(`jvm=clojure,"-J-Xmx2g"`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "jvm" {
		t.Fatalf("name = %q, want jvm", got.Name)
	}
	wantCommand := []string{"clojure", "-J-Xmx2g"}
	if len(got.Command) != len(wantCommand) {
		t.Fatalf("command = %#v, want %#v", got.Command, wantCommand)
	}
	for i := range wantCommand {
		if got.Command[i] != wantCommand[i] {
			t.Fatalf("command = %#v, want %#v", got.Command, wantCommand)
		}
	}
}

func TestParseRuntimeRejectsInvalidSpecs(t *testing.T) {
	for _, input := range []string{"", "name", "=command", "name="} {
		if _, err := parseRuntime(input); err == nil {
			t.Errorf("parseRuntime(%q) unexpectedly succeeded", input)
		}
	}
}

func TestManifestIsValid(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	path := filepath.Join(filepath.Dir(file), "manifest.json")
	suite, _, err := readManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(suite.Workloads) < 16 {
		t.Fatalf("manifest has %d workloads, want at least 16", len(suite.Workloads))
	}
}

func TestSelectWorkloadsMatchesNamesAndTags(t *testing.T) {
	workloads := []workload{
		{Name: "trees", Tags: []string{"records", "allocation"}},
		{Name: "numeric", Tags: []string{"floating-point"}},
	}
	selected := selectWorkloads(workloads, regexp.MustCompile("records|numeric"))
	if len(selected) != 2 {
		t.Fatalf("selected %#v, want both workloads", selected)
	}
}

func TestDriverSource(t *testing.T) {
	got := driverSource("/tmp/a benchmark.clj", "bench.example")
	for _, expected := range []string{
		`(load-file "/tmp/a benchmark.clj")`,
		`(in-ns 'bench.example)`,
		`(println (pr-str (run)))`,
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("driver does not contain %q:\n%s", expected, got)
		}
	}
}

func TestMedian(t *testing.T) {
	if got := median([]time.Duration{9, 1, 5}); got != 5 {
		t.Fatalf("odd median = %s, want 5ns", got)
	}
	if got := median([]time.Duration{9, 1, 5, 3}); got != 4 {
		t.Fatalf("even median = %s, want 4ns", got)
	}
}

func TestPercentile(t *testing.T) {
	values := []time.Duration{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if got := percentile(values, 0.95); got != 100 {
		t.Fatalf("p95 = %s, want 100ns", got)
	}
}
