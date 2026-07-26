package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	yamlstarRepository = "https://github.com/yaml/yamlstar.git"
	yamlstarRevision   = "64d1d0dd786877d2f0d8a96ccf7cd7905a8dc895"
)

func main() {
	rootFlag := flag.String("root", "", "Glojure repository root (default: auto-detect)")
	yamlstarRootFlag := flag.String("yamlstar-root", "", "local YAMLStar checkout at the benchmark revision (default: clone pinned revision)")
	gljFlag := flag.String("glojure", "", "development glj executable (default: build current checkout)")
	countFlag := flag.Int("count", 5, "Go benchmark repetition count")
	benchtimeFlag := flag.String("benchtime", "2s", "Go benchmark duration")
	benchFlag := flag.String("bench", ".", "Go benchmark name regexp")
	keepFlag := flag.Bool("keep", false, "retain the generated temporary module")
	flag.Parse()

	if *countFlag < 1 {
		fatalf("-count must be at least 1")
	}

	root := *rootFlag
	if root == "" {
		root = repositoryRoot()
	}
	root = mustAbs(root)

	temp, err := os.MkdirTemp("", "glojure-yamlstar-bench-")
	check(err)
	if *keepFlag {
		fmt.Fprintf(os.Stderr, "temporary benchmark module: %s\n", temp)
	} else {
		defer os.RemoveAll(temp)
	}

	yamlstarRoot := *yamlstarRootFlag
	if yamlstarRoot == "" {
		yamlstarRoot = cloneYAMLStar(temp)
	} else {
		yamlstarRoot = mustAbs(yamlstarRoot)
	}

	sourceRoot := filepath.Join(temp, "src")
	copyYAMLStarSources(yamlstarRoot, sourceRoot)
	applyCompatibilityRewrites(sourceRoot)
	copyFile(
		filepath.Join(root, "benchmark", "yamlstar", "fixture", "benchmark.glj"),
		filepath.Join(sourceRoot, "bench", "yamlstar.glj"),
	)

	glj := *gljFlag
	if glj == "" {
		glj = filepath.Join(temp, "glj")
		run(root, withGoCache(temp),
			"go", "build", "-tags", "glj_no_aot_stdlib", "-o", glj, "./cmd/glj")
	} else {
		glj = mustAbs(glj)
	}

	writeModule(root, temp)
	compileBenchmark(root, temp, sourceRoot, glj)
	writeGoBenchmark(temp)
	run(temp, withGoCache(temp), "go", "mod", "tidy")
	run(temp, withGoCache(temp), "go", "test", "-run", "^TestYAMLStarSemantics$", ".")

	args := []string{
		"test",
		"-run", "^$",
		"-bench", *benchFlag,
		"-benchmem",
		"-count", strconv.Itoa(*countFlag),
		"-benchtime", *benchtimeFlag,
		".",
	}
	run(temp, withGoCache(temp), "go", args...)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate benchmark source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func cloneYAMLStar(temp string) string {
	target := filepath.Join(temp, "yamlstar")
	run(temp, nil, "git", "clone", "--quiet", yamlstarRepository, target)
	run(target, nil, "git", "checkout", "--quiet", yamlstarRevision)
	return target
}

func copyYAMLStarSources(yamlstarRoot, sourceRoot string) {
	upstream := filepath.Join(yamlstarRoot, "core", "src")
	required := []string{
		filepath.Join("yamlstar", "api.clj"),
		filepath.Join("yamlstar", "parser", "grammar.clj"),
	}
	for _, name := range required {
		if info, err := os.Stat(filepath.Join(upstream, name)); err != nil || info.IsDir() {
			fatalf(
				"%s does not contain the source-based YAMLStar parser; check out %s",
				yamlstarRoot,
				yamlstarRevision,
			)
		}
	}

	err := filepath.WalkDir(upstream, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(upstream, path)
		if err != nil {
			return err
		}
		target := filepath.Join(sourceRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFileError(path, target)
	})
	check(err)
}

func applyCompatibilityRewrites(sourceRoot string) {
	replaceOnce(
		filepath.Join(sourceRoot, "yamlstar", "parser.clj"),
		"(catch Exception e",
		"(catch go/any e",
	)

	replaceOnce(
		filepath.Join(sourceRoot, "yamlstar", "parser", "parser.cljc"),
		`:glj (let [[r _] (unicode:utf8.DecodeRuneInString remaining)]
                        (when (> (int r) 65535)
                          (swap! (:pos p) inc))))`,
		`:glj (when (> (int (first remaining)) 65535)
                        (swap! (:pos p) inc)))`,
	)
}

func replaceOnce(path, old, replacement string) {
	data, err := os.ReadFile(path)
	check(err)
	if bytes.Count(data, []byte(old)) != 1 {
		fatalf("expected exactly one compatibility rewrite target in %s", path)
	}
	data = bytes.Replace(data, []byte(old), []byte(replacement), 1)
	check(os.WriteFile(path, data, 0o644))
}

func writeModule(root, temp string) {
	module := fmt.Sprintf(`module glojure-yamlstar-bench

go 1.24

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, filepath.ToSlash(root))
	check(os.WriteFile(filepath.Join(temp, "go.mod"), []byte(module), 0o644))
	copyFile(filepath.Join(root, "go.sum"), filepath.Join(temp, "go.sum"))
}

func compileBenchmark(root, temp, sourceRoot, glj string) {
	env := append(os.Environ(),
		"GLJ_CLASSPATH="+sourceRoot,
		"GLOJURE_STDLIB_PATH="+filepath.Join(root, "pkg", "stdlib"),
		"GLOJURE_USE_AOT=false",
	)
	run(temp, env, glj, "-e", "(compile 'bench.yamlstar)")
}

func writeGoBenchmark(temp string) {
	source := `package yamlstarbench

import (
	"sync"
	"testing"

	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	benchmark "glojure-yamlstar-bench/src/bench/yamlstar"
	api "glojure-yamlstar-bench/src/yamlstar/api"
	composer "glojure-yamlstar-bench/src/yamlstar/composer"
	constructor "glojure-yamlstar-bench/src/yamlstar/constructor"
	desolver "glojure-yamlstar-bench/src/yamlstar/desolver"
	emitter "glojure-yamlstar-bench/src/yamlstar/emitter"
	numbers "glojure-yamlstar-bench/src/yamlstar/numbers"
	parser "glojure-yamlstar-bench/src/yamlstar/parser"
	grammar "glojure-yamlstar-bench/src/yamlstar/parser/grammar"
	parserImpl "glojure-yamlstar-bench/src/yamlstar/parser/parser"
	prelude "glojure-yamlstar-bench/src/yamlstar/parser/prelude"
	receiver "glojure-yamlstar-bench/src/yamlstar/parser/receiver"
	representer "glojure-yamlstar-bench/src/yamlstar/representer"
	resolver "glojure-yamlstar-bench/src/yamlstar/resolver"
	serializer "glojure-yamlstar-bench/src/yamlstar/serializer"
)

var benchmarkResult any
var loadOnce sync.Once

type yamlstarCase struct {
	name     string
	parseVar string
	loadVar  string
	expectedVar string
}

var yamlstarCases = []yamlstarCase{
	{name: "scalar", parseVar: "parse-scalar", loadVar: "load-scalar", expectedVar: "expected-scalar"},
	{name: "mapping", parseVar: "parse-mapping", loadVar: "load-mapping", expectedVar: "expected-mapping"},
	{name: "nested", parseVar: "parse-nested", loadVar: "load-nested", expectedVar: "expected-nested"},
	{name: "types", parseVar: "parse-types", loadVar: "load-types", expectedVar: "expected-types"},
}

func TestYAMLStarSemantics(t *testing.T) {
	loadYAMLStar()
	ns := lang.FindNamespace(lang.NewSymbol("bench.yamlstar"))
	for _, tc := range yamlstarCases {
		parse := benchmarkFunction(t, ns, tc.parseVar)
		events := parse.Invoke()
		if events == nil || lang.Count(events) == 0 {
			t.Errorf("%s returned no parser events", tc.parseVar)
		}

		load := benchmarkFunction(t, ns, tc.loadVar)
		expected := ns.FindInternedVar(lang.NewSymbol(tc.expectedVar)).Get()
		if got := load.Invoke(); !lang.Equals(got, expected) {
			t.Errorf("%s returned %s, want %s",
				tc.loadVar, lang.PrintString(got), lang.PrintString(expected))
		}
	}
}

func BenchmarkParse(b *testing.B) {
	loadYAMLStar()
	ns := lang.FindNamespace(lang.NewSymbol("bench.yamlstar"))
	for _, tc := range yamlstarCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			run := benchmarkFunction(b, ns, tc.parseVar)
			benchmarkResult = run.Invoke()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkResult = run.Invoke()
			}
		})
	}
}

func BenchmarkLoad(b *testing.B) {
	loadYAMLStar()
	ns := lang.FindNamespace(lang.NewSymbol("bench.yamlstar"))
	for _, tc := range yamlstarCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			run := benchmarkFunction(b, ns, tc.loadVar)
			benchmarkResult = run.Invoke()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkResult = run.Invoke()
			}
		})
	}
}

func loadYAMLStar() {
	loadOnce.Do(func() {
		prelude.LoadNS()
		parserImpl.LoadNS()
		grammar.LoadNS()
		receiver.LoadNS()
		parser.LoadNS()
		composer.LoadNS()
		resolver.LoadNS()
		numbers.LoadNS()
		constructor.LoadNS()
		representer.LoadNS()
		desolver.LoadNS()
		serializer.LoadNS()
		emitter.LoadNS()
		api.LoadNS()
		benchmark.LoadNS()
	})
}

type fataler interface {
	Helper()
	Fatalf(string, ...any)
}

func benchmarkFunction(t fataler, ns *lang.Namespace, name string) lang.IFn {
	t.Helper()
	vr := ns.FindInternedVar(lang.NewSymbol(name))
	if vr == nil {
		t.Fatalf("missing benchmark Var %s", name)
	}
	fn, ok := vr.Get().(lang.IFn)
	if !ok {
		t.Fatalf("%s has type %T, want lang.IFn", name, vr.Get())
	}
	return fn
}
`
	formatted, err := format.Source([]byte(source))
	check(err)
	check(os.WriteFile(filepath.Join(temp, "yamlstar_benchmark_test.go"), formatted, 0o644))
}

func copyFile(source, target string) {
	check(copyFileError(source, target))
}

func copyFileError(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func withGoCache(temp string) []string {
	return append(os.Environ(), "GOCACHE="+filepath.Join(temp, "go-cache"))
}

func run(dir string, env []string, name string, args ...string) {
	fmt.Fprintf(os.Stderr, "+ %s\n", strings.Join(append([]string{name}, args...), " "))
	command := exec.Command(name, args...)
	command.Dir = dir
	if env != nil {
		command.Env = env
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	check(command.Run())
}

func mustAbs(path string) string {
	absolute, err := filepath.Abs(path)
	check(err)
	return absolute
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "yamlstar benchmark: "+format+"\n", args...)
	os.Exit(2)
}
