// Command aot compiles every portable suite workload with the current Glojure
// compiler and runs in-process Go benchmarks against the generated functions.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type manifest struct {
	Version   int        `json:"version"`
	Workloads []workload `json:"workloads"`
}

type workload struct {
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	File       string   `json:"file"`
	Expected   string   `json:"expected"`
	Tags       []string `json:"tags"`
	alias      string
	importPath string
}

func main() {
	rootFlag := flag.String("root", "", "Glojure repository root (default: auto-detect)")
	manifestFlag := flag.String("manifest", "", "suite manifest (default: benchmark/suite/manifest.json)")
	gljFlag := flag.String("glojure", "", "bootstrap glj executable (default: build current checkout)")
	goFlag := flag.String("go", "go", "Go command")
	countFlag := flag.Int("count", 5, "go test benchmark repetition count")
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
	manifestPath := *manifestFlag
	if manifestPath == "" {
		manifestPath = filepath.Join(root, "benchmark", "suite", "manifest.json")
	}
	suite, manifestDir := readManifest(manifestPath)

	temp, err := os.MkdirTemp("", "glojure-portable-aot-")
	check(err)
	if *keepFlag {
		fmt.Fprintf(os.Stderr, "temporary AOT module: %s\n", temp)
	} else {
		defer os.RemoveAll(temp)
	}

	copySources(temp, manifestDir, suite.Workloads)
	writeModule(root, temp)

	glj := *gljFlag
	if glj == "" {
		glj = filepath.Join(temp, "glj-bootstrap")
		run(root, nil, *goFlag,
			"build",
			"-tags", "glj_no_aot_stdlib",
			"-o", glj,
			"./cmd/glj",
		)
	} else {
		glj = mustAbs(glj)
	}
	compileSources(root, temp, glj, suite.Workloads)
	writeBenchmark(temp, suite.Workloads)

	run(temp, nil, *goFlag, "mod", "tidy")
	run(temp, nil, *goFlag,
		"test",
		"-run", "^TestCorrectness$",
		"-bench", *benchFlag,
		"-benchmem",
		"-count", fmt.Sprint(*countFlag),
		"-benchtime", *benchtimeFlag,
		".",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate benchmark suite")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readManifest(path string) (manifest, string) {
	path = mustAbs(path)
	data, err := os.ReadFile(path)
	check(err)
	var suite manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	check(decoder.Decode(&suite))
	if suite.Version != 1 {
		fatalf("unsupported manifest version %d", suite.Version)
	}
	if len(suite.Workloads) == 0 {
		fatalf("manifest contains no workloads")
	}
	sort.Slice(suite.Workloads, func(i, j int) bool {
		return suite.Workloads[i].Name < suite.Workloads[j].Name
	})
	aliases := make(map[string]bool, len(suite.Workloads))
	for i := range suite.Workloads {
		workload := &suite.Workloads[i]
		if workload.Name == "" || workload.Namespace == "" ||
			workload.File == "" || workload.Expected == "" {
			fatalf("incomplete workload at manifest index %d", i)
		}
		workload.alias = goIdentifier(workload.Name)
		if aliases[workload.alias] {
			fatalf("workloads have colliding Go identifier %q", workload.alias)
		}
		aliases[workload.alias] = true
		workload.importPath = strings.ReplaceAll(
			strings.ReplaceAll(workload.Namespace, ".", "/"),
			"-",
			"_",
		)
	}
	return suite, filepath.Dir(path)
}

func copySources(temp, manifestDir string, workloads []workload) {
	for _, workload := range workloads {
		source := filepath.Join(manifestDir, workload.File)
		data, err := os.ReadFile(source)
		check(err)
		target := filepath.Join(temp, "src", workload.importPath+".clj")
		check(os.MkdirAll(filepath.Dir(target), 0o755))
		check(os.WriteFile(target, data, 0o644))
	}
}

func writeModule(root, temp string) {
	source := fmt.Sprintf(`module glojure-portable-aot

go 1.24

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, filepath.ToSlash(root))
	check(os.WriteFile(filepath.Join(temp, "go.mod"), []byte(source), 0o644))
	data, err := os.ReadFile(filepath.Join(root, "go.sum"))
	check(err)
	check(os.WriteFile(filepath.Join(temp, "go.sum"), data, 0o644))
}

func compileSources(root, temp, glj string, workloads []workload) {
	env := append(os.Environ(),
		"GLJ_CLASSPATH="+filepath.Join(temp, "src"),
		"GLOJURE_STDLIB_PATH="+filepath.Join(root, "pkg", "stdlib"),
		"GLOJURE_USE_AOT=false",
	)
	// Compile each independent workload in a fresh process. Besides bounding
	// compiler state and memory, this makes a failure identify the exact
	// workload instead of aborting an opaque batch expression.
	for _, workload := range workloads {
		run(temp, env, glj, "-e", "(compile '"+workload.Namespace+")")
	}
}

func writeBenchmark(temp string, workloads []workload) {
	var source bytes.Buffer
	source.WriteString("package portableaot\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"testing\"\n\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/glj\"\n")
	source.WriteString("\t\"github.com/glojurelang/glojure/pkg/lang\"\n")
	for _, workload := range workloads {
		fmt.Fprintf(
			&source,
			"\t%s \"glojure-portable-aot/src/%s\"\n",
			workload.alias,
			workload.importPath,
		)
	}
	source.WriteString(")\n\n")
	source.WriteString("var benchmarkResult any\n\n")
	source.WriteString("func TestCorrectness(t *testing.T) {\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source, "\t%s.LoadNS()\n", workload.alias)
		fmt.Fprintf(
			&source,
			"\tns%s := lang.FindNamespace(lang.NewSymbol(%q))\n",
			workload.alias,
			workload.Namespace,
		)
		fmt.Fprintf(
			&source,
			"\trun%s := ns%s.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n",
			workload.alias,
			workload.alias,
		)
		fmt.Fprintf(
			&source,
			"\tif got := lang.PrintString(run%s.Invoke()); got != %q {\n",
			workload.alias,
			workload.Expected,
		)
		fmt.Fprintf(
			&source,
			"\t\tt.Errorf(%q, got)\n",
			workload.Name+" returned %s",
		)
		source.WriteString("\t}\n")
	}
	source.WriteString("}\n\n")
	for _, workload := range workloads {
		fmt.Fprintf(
			&source,
			"func Benchmark%s(b *testing.B) {\n",
			exportedIdentifier(workload.Name),
		)
		fmt.Fprintf(&source, "\t%s.LoadNS()\n", workload.alias)
		fmt.Fprintf(
			&source,
			"\tns := lang.FindNamespace(lang.NewSymbol(%q))\n",
			workload.Namespace,
		)
		source.WriteString(
			"\trun := ns.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n",
		)
		source.WriteString("\tbenchmarkResult = run.Invoke()\n")
		source.WriteString("\tb.ReportAllocs()\n")
		source.WriteString("\tb.ResetTimer()\n")
		source.WriteString("\tfor i := 0; i < b.N; i++ {\n")
		source.WriteString("\t\tbenchmarkResult = run.Invoke()\n")
		source.WriteString("\t}\n")
		source.WriteString("}\n\n")
	}
	formatted, err := format.Source(source.Bytes())
	if err != nil {
		_ = os.WriteFile(filepath.Join(temp, "benchmark.invalid.go"), source.Bytes(), 0o644)
		fatalf("format generated benchmark: %v", err)
	}
	check(os.WriteFile(
		filepath.Join(temp, "benchmark_test.go"),
		formatted,
		0o644,
	))
}

func goIdentifier(value string) string {
	var result strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') {
			result.WriteRune(char)
		} else {
			result.WriteByte('_')
		}
	}
	return result.String()
}

func exportedIdentifier(value string) string {
	var result strings.Builder
	upper := true
	for _, char := range value {
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
