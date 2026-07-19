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
