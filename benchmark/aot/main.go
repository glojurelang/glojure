package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type fixture struct {
	fileName string
	nsName   string
	alias    string
}

func main() {
	rootFlag := flag.String("root", "", "Glojure repository root (default: auto-detect)")
	gljFlag := flag.String("glojure", "", "development glj executable (default: build current checkout)")
	count := flag.Int("count", 5, "go test benchmark repetition count")
	benchtime := flag.String("benchtime", "2s", "Go benchmark duration")
	filter := flag.String("bench", ".", "Go benchmark name regexp")
	keep := flag.Bool("keep", false, "retain the generated temporary module")
	flag.Parse()

	root := *rootFlag
	if root == "" {
		root = repositoryRoot()
	}
	root = mustAbs(root)

	temp, err := os.MkdirTemp("", "glojure-aot-bench-")
	check(err)
	if !*keep {
		defer os.RemoveAll(temp)
	} else {
		fmt.Fprintf(os.Stderr, "temporary benchmark module: %s\n", temp)
	}

	fixtures := copyFixtures(root, temp)
	glj := *gljFlag
	if glj == "" {
		glj = filepath.Join(temp, "glj")
		run(root, nil, "go", "build", "-tags", "glj_no_aot_stdlib", "-o", glj, "./cmd/glj")
	} else {
		glj = mustAbs(glj)
	}

	writeModule(root, temp)
	compileFixtures(root, temp, glj, fixtures)
	writeBenchmark(temp, fixtures)
	run(temp, nil, "go", "mod", "tidy")
	run(temp, nil, "go", "test", "-run", "^TestAOTSemantics$", "./...")

	args := []string{
		"test",
		"-run", "^$",
		"-bench", *filter,
		"-benchmem",
		"-count", strconv.Itoa(*count),
		"-benchtime", *benchtime,
		"./...",
	}
	run(temp, nil, "go", args...)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate benchmark source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyFixtures(root, temp string) []fixture {
	sourceDir := filepath.Join(root, "benchmark", "aot", "fixtures")
	entries, err := os.ReadDir(sourceDir)
	check(err)

	var fixtures []fixture
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".glj" {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".glj")
		nsName := "bench." + base
		alias := strings.ReplaceAll(base, "-", "_")
		resourceName := strings.ReplaceAll(entry.Name(), "-", "_")
		target := filepath.Join(temp, "src", "bench", resourceName)
		check(os.MkdirAll(filepath.Dir(target), 0o755))
		data, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		check(err)
		check(os.WriteFile(target, data, 0o644))
		fixtures = append(fixtures, fixture{
			fileName: entry.Name(),
			nsName:   nsName,
			alias:    alias,
		})
	}
	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].fileName < fixtures[j].fileName
	})
	if len(fixtures) == 0 {
		panic("no AOT benchmark fixtures found")
	}
	return fixtures
}

func writeModule(root, temp string) {
	module := fmt.Sprintf(`module glojure-aot-bench

go 1.24

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, filepath.ToSlash(root))
	check(os.WriteFile(filepath.Join(temp, "go.mod"), []byte(module), 0o644))
	sums, err := os.ReadFile(filepath.Join(root, "go.sum"))
	check(err)
	check(os.WriteFile(filepath.Join(temp, "go.sum"), sums, 0o644))
}

func compileFixtures(root, temp, glj string, fixtures []fixture) {
	forms := make([]string, len(fixtures))
	for i, fixture := range fixtures {
		forms[i] = "'" + fixture.nsName
	}
	expression := "(mapv compile [" + strings.Join(forms, " ") + "])"
	env := append(os.Environ(),
		"GLJ_CLASSPATH="+filepath.Join(temp, "src"),
		"GLOJURE_STDLIB_PATH="+filepath.Join(root, "pkg", "stdlib"),
		"GLOJURE_USE_AOT=false",
	)
	run(temp, env, glj, "-e", expression)
}

func writeBenchmark(temp string, fixtures []fixture) {
	var source bytes.Buffer
	source.WriteString("package aotbench\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"testing\"\n\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/glj\"\n")
	source.WriteString("\t\"github.com/glojurelang/glojure/pkg/lang\"\n")
	for _, fixture := range fixtures {
		fmt.Fprintf(&source, "\t%s \"glojure-aot-bench/src/bench/%s\"\n",
			fixture.alias, fixture.alias)
	}
	source.WriteString(")\n\n")
	source.WriteString("var benchmarkResult any\n\n")
	source.WriteString("func TestAOTSemantics(t *testing.T) {\n")
	for _, fixture := range fixtures {
		fmt.Fprintf(&source, "\t%s.LoadNS()\n", fixture.alias)
		fmt.Fprintf(&source, "\t%sNS := lang.FindNamespace(lang.NewSymbol(%q))\n",
			fixture.alias, fixture.nsName)
		fmt.Fprintf(&source, "\t%sRun := %sNS.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n",
			fixture.alias, fixture.alias)
		fmt.Fprintf(&source, "\t%sExpected := %sNS.FindInternedVar(lang.NewSymbol(\"expected\")).Get()\n",
			fixture.alias, fixture.alias)
		fmt.Fprintf(&source, "\tif got := %sRun.Invoke(); !lang.Equals(got, %sExpected) {\n",
			fixture.alias, fixture.alias)
		fmt.Fprintf(&source, "\t\tt.Fatalf(%q, got, %sExpected)\n",
			fixture.nsName+" run = %v, want %v", fixture.alias)
		source.WriteString("\t}\n")
	}
	source.WriteString("\tpolynomial := float_kernelNS.FindInternedVar(lang.NewSymbol(\"polynomial\"))\n")
	source.WriteString("\tcaller := float_kernelNS.FindInternedVar(lang.NewSymbol(\"call-polynomial\")).Get().(lang.IFn)\n")
	source.WriteString("\tfor _, input := range []any{float64(2), int64(2)} {\n")
	source.WriteString("\t\tif got := lang.Apply1(caller, input); !lang.Equals(got, float64(5)) {\n")
	source.WriteString("\t\t\tt.Fatalf(\"call-polynomial(%T(2)) = %v, want 5.0\", input, got)\n")
	source.WriteString("\t\t}\n")
	source.WriteString("\t}\n")
	source.WriteString("\toriginal := polynomial.Get()\n")
	source.WriteString("\tpolynomial.BindRoot(lang.FnFunc1(func(any) any { return float64(42) }))\n")
	source.WriteString("\tdefer polynomial.BindRoot(original)\n")
	source.WriteString("\tif got := lang.Apply1(caller, float64(2)); !lang.Equals(got, float64(42)) {\n")
	source.WriteString("\t\tt.Fatalf(\"call-polynomial ignored Var redefinition: got %v, want 42.0\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tpolynomial.BindRoot(original)\n")
	source.WriteString("\tincVar := lang.FindNamespace(lang.NewSymbol(\"clojure.core\")).FindInternedVar(lang.NewSymbol(\"inc\"))\n")
	source.WriteString("\toriginalInc := incVar.Get()\n")
	source.WriteString("\tincVar.BindRoot(lang.FnFunc1(func(value any) any { return value }))\n")
	source.WriteString("\tdefer incVar.BindRoot(originalInc)\n")
	source.WriteString("\tif got := reduce_pipelineRun.Invoke(); !lang.Equals(got, int64(250000000000)) {\n")
	source.WriteString("\t\tt.Fatalf(\"reduce pipeline ignored inc redefinition: got %v, want 250000000000\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tincVar.BindRoot(originalInc)\n")
	source.WriteString("\ttakeVar := lang.FindNamespace(lang.NewSymbol(\"clojure.core\")).FindInternedVar(lang.NewSymbol(\"take\"))\n")
	source.WriteString("\toriginalTake := takeVar.Get()\n")
	source.WriteString("\ttakeVar.BindRoot(lang.FnFunc2(func(any, any) any { return lang.NewVector(int64(7)) }))\n")
	source.WriteString("\tdefer takeVar.BindRoot(originalTake)\n")
	source.WriteString("\tif got := letgo_map_filterRun.Invoke(); !lang.Equals(got, int64(1313400)) {\n")
	source.WriteString("\t\tt.Fatalf(\"direct-linked map/filter pipeline changed after take redefinition: got %v, want 1313400\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tcountCaller := constant_arithmeticNS.FindInternedVar(lang.NewSymbol(\"count-input\")).Get().(lang.IFn)\n")
	source.WriteString("\tcountVar := lang.FindNamespace(lang.NewSymbol(\"clojure.core\")).FindInternedVar(lang.NewSymbol(\"count\"))\n")
	source.WriteString("\toriginalCount := countVar.Get()\n")
	source.WriteString("\tcountVar.BindRoot(lang.FnFunc1(func(any) any { return int64(99) }))\n")
	source.WriteString("\tdefer countVar.BindRoot(originalCount)\n")
	source.WriteString("\tif got := lang.Apply1(countCaller, lang.NewVector(1, 2, 3)); !lang.Equals(got, int64(3)) {\n")
	source.WriteString("\t\tt.Fatalf(\"direct-linked count changed after redefinition: got %v, want 3\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tlocalCounter := constant_arithmeticNS.FindInternedVar(lang.NewSymbol(\"local-counter\")).Get().(lang.IFn)\n")
	source.WriteString("\tif got := localCounter.Invoke(); !lang.Equals(got, int64(42)) {\n")
	source.WriteString("\t\tt.Fatalf(\"scalar-replaced local atom = %v, want 42\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("\townedUpdate := owned_vectorNS.FindInternedVar(lang.NewSymbol(\"update-all\")).Get().(lang.IFn)\n")
	source.WriteString("\townedValues := owned_vectorNS.FindInternedVar(lang.NewSymbol(\"values\")).Get()\n")
	source.WriteString("\townedPairs := owned_vectorNS.FindInternedVar(lang.NewSymbol(\"pairs\")).Get().(*lang.Vector)\n")
	source.WriteString("\townedExpected := owned_vectorNS.FindInternedVar(lang.NewSymbol(\"expected\")).Get()\n")
	source.WriteString("\tpairList := lang.NewList(ownedPairs.Nth(0), ownedPairs.Nth(1))\n")
	source.WriteString("\tif got := lang.Apply3(ownedUpdate, ownedValues, pairList, int64(10)); !lang.Equals(got, ownedExpected) {\n")
	source.WriteString("\t\tt.Fatalf(\"owned-vector non-Indexed fallback = %v, want %v\", got, ownedExpected)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tpreserveVersion := owned_vectorNS.FindInternedVar(lang.NewSymbol(\"preserve-earlier-version\")).Get().(lang.IFn)\n")
	source.WriteString("\tif got := lang.Apply1(preserveVersion, lang.NewVector(int64(1), int64(2))); !lang.Equals(got, lang.NewVector(int64(20), int64(1))) {\n")
	source.WriteString("\t\tt.Fatalf(\"owned-vector earlier version = %v, want [20 1]\", got)\n")
	source.WriteString("\t}\n")
	source.WriteString("}\n\n")
	for _, fixture := range fixtures {
		benchmarkName := exportedName(strings.TrimPrefix(fixture.nsName, "bench."))
		fmt.Fprintf(&source, "func Benchmark%s(b *testing.B) {\n", benchmarkName)
		fmt.Fprintf(&source, "\t%s.LoadNS()\n", fixture.alias)
		fmt.Fprintf(&source, "\tns := lang.FindNamespace(lang.NewSymbol(%q))\n", fixture.nsName)
		source.WriteString("\trun := ns.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n")
		source.WriteString("\tbenchmarkResult = run.Invoke()\n")
		source.WriteString("\tb.ReportAllocs()\n")
		source.WriteString("\tb.ResetTimer()\n")
		source.WriteString("\tfor i := 0; i < b.N; i++ {\n")
		source.WriteString("\t\tbenchmarkResult = run.Invoke()\n")
		source.WriteString("\t}\n")
		source.WriteString("}\n\n")
	}
	check(os.WriteFile(
		filepath.Join(temp, "aot_benchmark_test.go"),
		source.Bytes(),
		0o644,
	))
}

func exportedName(name string) string {
	var result strings.Builder
	upper := true
	for _, char := range name {
		if char == '-' || char == '_' || char == '.' {
			upper = true
			continue
		}
		if upper && char >= 'a' && char <= 'z' {
			char -= 'a' - 'A'
		}
		result.WriteRune(char)
		upper = false
	}
	return result.String()
}

func mustAbs(path string) string {
	absolute, err := filepath.Abs(path)
	check(err)
	return absolute
}

func run(dir string, env []string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "+ %s\n", strings.Join(cmd.Args, " "))
	check(cmd.Run())
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
