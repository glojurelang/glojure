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
	"slices"
	"strconv"
	"strings"
	"time"
)

type workload struct {
	name     string
	fileName string
	nsName   string
	alias    string
	expected int64
}

var workloads = []workload{
	{
		name:     "batched-fib",
		fileName: "batched-fib.clj",
		nsName:   "bench.long.batched-fib",
		alias:    "batchedFib",
		expected: 184549300,
	},
	{
		name:     "compute-bound",
		fileName: "compute-bound.clj",
		nsName:   "bench.long.compute-bound",
		alias:    "computeBound",
		expected: 1249925001000000,
	},
	{
		name:     "event-analytics",
		fileName: "event-analytics.clj",
		nsName:   "bench.long.event-analytics",
		alias:    "eventAnalytics",
		expected: 5771995737,
	},
	{
		name:     "game-of-life",
		fileName: "game-of-life.clj",
		nsName:   "bench.long.game-of-life",
		alias:    "gameOfLife",
		expected: 2187312,
	},
}

func main() {
	rootFlag := flag.String("root", "", "Glojure repository root (default: auto-detect)")
	letGoRootFlag := flag.String("let-go-root", "../let-go", "let-go repository root")
	glojureAOTFlag := flag.String("glojure-aot", "", "reuse a Glojure long-AOT executable")
	letGoAOTFlag := flag.String("let-go-aot", "", "reuse a let-go long-AOT executable")
	glojureGoFlag := flag.String("glojure-go", "go", "Go command used to build Glojure")
	letGoGoFlag := flag.String("let-go-go", "go", "Go command used to build let-go")
	runsFlag := flag.Int("runs", 5, "timed process runs per runtime and workload")
	keepFlag := flag.Bool("keep", false, "retain generated modules and executables")
	flag.Parse()

	if *runsFlag < 1 {
		fatalf("-runs must be at least 1")
	}

	root := *rootFlag
	if root == "" {
		root = repositoryRoot()
	}
	root = mustAbs(root)
	letGoRoot := mustAbs(*letGoRootFlag)

	temp, err := os.MkdirTemp("", "glojure-long-aot-")
	check(err)
	if *keepFlag {
		fmt.Fprintf(os.Stderr, "temporary benchmark directory: %s\n", temp)
	} else {
		defer os.RemoveAll(temp)
	}

	glojureAOT := reusableExecutable(*glojureAOTFlag)
	if glojureAOT == "" {
		glojureAOT = buildGlojure(root, temp, *glojureGoFlag)
	}
	letGoAOT := reusableExecutable(*letGoAOTFlag)
	if letGoAOT == "" {
		letGoAOT = buildLetGo(root, letGoRoot, temp, *letGoGoFlag)
	}

	compare(glojureAOT, letGoAOT, *runsFlag)
	printSizes(glojureAOT, letGoAOT)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate benchmark source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func reusableExecutable(path string) string {
	if path == "" {
		return ""
	}
	path = mustAbs(path)
	info, err := os.Stat(path)
	check(err)
	if info.IsDir() || info.Mode()&0o111 == 0 {
		fatalf("%s is not an executable file", path)
	}
	return path
}

func buildGlojure(root, temp, goCommand string) string {
	module := filepath.Join(temp, "glojure")
	sourceRoot := filepath.Join(module, "src")
	copyGlojureFixtures(root, sourceRoot)

	bootstrap := filepath.Join(temp, "glj")
	run(root, nil, goCommand,
		"build", "-tags", "glj_no_aot_stdlib", "-o", bootstrap, "./cmd/glj")

	namespaces := make([]string, len(workloads))
	for i, workload := range workloads {
		namespaces[i] = "'" + workload.nsName
	}
	expression := "(mapv compile [" + strings.Join(namespaces, " ") + "])"
	env := append(os.Environ(),
		"GLJ_CLASSPATH="+sourceRoot,
		"GLOJURE_STDLIB_PATH="+filepath.Join(root, "pkg", "stdlib"),
		"GLOJURE_USE_AOT=false",
	)
	run(module, env, bootstrap, "-e", expression)

	writeFile(filepath.Join(module, "go.mod"), fmt.Sprintf(`module long-aot

go 1.24

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, filepath.ToSlash(root)))
	copyFile(filepath.Join(root, "go.sum"), filepath.Join(module, "go.sum"))
	writeFormattedGo(filepath.Join(module, "main.go"), glojureMainSource())

	run(module, nil, goCommand, "mod", "tidy")
	output := filepath.Join(temp, "glojure-long-aot")
	run(module, nil, goCommand,
		"build", "-tags", "glj_aot_runtime",
		"-ldflags", "-s -w", "-o", output, ".")
	return output
}

func copyGlojureFixtures(root, sourceRoot string) {
	fixtureRoot := filepath.Join(root, "benchmark", "aot", "long", "fixtures")
	for _, workload := range workloads {
		resourceName := strings.ReplaceAll(workload.fileName, "-", "_")
		target := filepath.Join(sourceRoot, "bench", "long", resourceName)
		copyFile(filepath.Join(fixtureRoot, workload.fileName), target)
	}
}

func glojureMainSource() string {
	var source bytes.Buffer
	source.WriteString("package main\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"fmt\"\n")
	source.WriteString("\t\"os\"\n\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/glj\"\n")
	source.WriteString("\t\"github.com/glojurelang/glojure/pkg/lang\"\n\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source,
			"\t%s \"long-aot/src/bench/long/%s\"\n",
			workload.alias,
			strings.ReplaceAll(strings.TrimSuffix(workload.fileName, ".clj"), "-", "_"),
		)
	}
	source.WriteString(")\n\n")
	source.WriteString("var expected = map[string]int64{\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source, "\t%q: %d,\n", workload.name, workload.expected)
	}
	source.WriteString("}\n\n")
	source.WriteString("func main() {\n")
	source.WriteString("\tif len(os.Args) != 2 {\n")
	source.WriteString("\t\tpanic(\"expected workload name\")\n")
	source.WriteString("\t}\n")
	source.WriteString("\tname := os.Args[1]\n")
	source.WriteString("\twant, ok := expected[name]\n")
	source.WriteString("\tif !ok {\n")
	source.WriteString("\t\tpanic(\"unknown workload\")\n")
	source.WriteString("\t}\n")
	source.WriteString("\tswitch name {\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source, "\tcase %q:\n\t\t%s.LoadNS()\n",
			workload.name, workload.alias)
	}
	source.WriteString("\t}\n")
	source.WriteString("\tns := lang.FindNamespace(lang.NewSymbol(\"bench.long.\" + name))\n")
	source.WriteString("\trun := ns.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n")
	source.WriteString("\tresult := run.Invoke()\n")
	source.WriteString("\tif !lang.Equals(result, want) {\n")
	source.WriteString("\t\tpanic(fmt.Sprintf(\"%s returned %v, want %d\", name, result, want))\n")
	source.WriteString("\t}\n")
	source.WriteString("\tfmt.Println(result)\n")
	source.WriteString("}\n")
	return source.String()
}

func buildLetGo(root, letGoRoot, temp, goCommand string) string {
	clone := filepath.Join(temp, "let-go")
	run(temp, nil, "git", "clone", "--quiet", "--local", "--no-hardlinks",
		letGoRoot, clone)

	compiler := filepath.Join(clone, "lg")
	run(clone, nil, goCommand, "build", "-o", compiler, ".")
	run(clone, nil, goCommand,
		"run", "-tags", "bootstrap", "./cmd/lgbgen", "--target=go")

	module := filepath.Join(temp, "let-go-app")
	generated := filepath.Join(module, "generated")
	fixtureRoot := filepath.Join(root, "benchmark", "aot", "long", "fixtures")
	args := []string{
		"scripts/lg-compile",
		generated,
		"long-aot/generated",
	}
	for _, workload := range workloads {
		args = append(args, filepath.Join(fixtureRoot, workload.fileName))
	}
	run(clone, nil, compiler, args...)

	goVersion := moduleGoVersion(filepath.Join(clone, "go.mod"))
	writeFile(filepath.Join(module, "go.mod"), fmt.Sprintf(`module long-aot

go %s

require github.com/nooga/let-go v0.0.0

replace github.com/nooga/let-go => %s
`, goVersion, filepath.ToSlash(clone)))
	copyFile(filepath.Join(clone, "go.sum"), filepath.Join(module, "go.sum"))
	writeFormattedGo(filepath.Join(module, "main.go"), letGoMainSource())

	// The application packages register their own native overrides. Copying
	// let-go's generated import wireup also links the native gogen_ir core;
	// without it, dynamic calls from the application would silently hit the VM.
	copyFile(
		filepath.Join(clone, "lg_gogen_ir.go"),
		filepath.Join(module, "core_gogen_ir.go"),
	)

	run(module, nil, goCommand, "mod", "tidy")
	output := filepath.Join(temp, "let-go-long-aot")
	run(module, nil, goCommand,
		"build", "-tags", "gogen_ir", "-ldflags", "-s -w", "-o", output, ".")
	return output
}

func letGoMainSource() string {
	var source bytes.Buffer
	source.WriteString("package main\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"fmt\"\n")
	source.WriteString("\t\"os\"\n\n")
	source.WriteString("\t\"github.com/nooga/let-go/pkg/rt\"\n")
	source.WriteString("\t_ \"github.com/nooga/let-go/pkg/rt/corefns\"\n")
	source.WriteString("\t\"github.com/nooga/let-go/pkg/vm\"\n\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source,
			"\t_ \"long-aot/generated/bench/long/%s\"\n",
			strings.ReplaceAll(workload.name, "-", "_"),
		)
	}
	source.WriteString(")\n\n")
	source.WriteString("var expected = map[string]string{\n")
	for _, workload := range workloads {
		fmt.Fprintf(&source, "\t%q: %q,\n",
			workload.name, strconv.FormatInt(workload.expected, 10))
	}
	source.WriteString("}\n\n")
	source.WriteString("func main() {\n")
	source.WriteString("\tif len(os.Args) != 2 {\n")
	source.WriteString("\t\tpanic(\"expected workload name\")\n")
	source.WriteString("\t}\n")
	source.WriteString("\tname := os.Args[1]\n")
	source.WriteString("\twant, ok := expected[name]\n")
	source.WriteString("\tif !ok {\n")
	source.WriteString("\t\tpanic(\"unknown workload\")\n")
	source.WriteString("\t}\n")
	source.WriteString("\tec, err := rt.BootCore()\n")
	source.WriteString("\tif err != nil {\n")
	source.WriteString("\t\tpanic(err)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tns := rt.LookupOrRegisterNS(\"bench.long.\" + name)\n")
	source.WriteString("\trt.ApplyGoOverrides(ns)\n")
	source.WriteString("\trunVar, ok := ns.Lookup(vm.Symbol(\"run\")).(*vm.Var)\n")
	source.WriteString("\tif !ok {\n")
	source.WriteString("\t\tpanic(\"missing run Var\")\n")
	source.WriteString("\t}\n")
	source.WriteString("\trun, ok := runVar.Deref().(*vm.NativeFn)\n")
	source.WriteString("\tif !ok {\n")
	source.WriteString("\t\tpanic(fmt.Sprintf(\"run was not AOT-lowered: %T\", runVar.Deref()))\n")
	source.WriteString("\t}\n")
	source.WriteString("\tresult, err := ec.Invoke(run, nil)\n")
	source.WriteString("\tif err != nil {\n")
	source.WriteString("\t\tpanic(err)\n")
	source.WriteString("\t}\n")
	source.WriteString("\trendered, err := vm.SafeString(result)\n")
	source.WriteString("\tif err != nil {\n")
	source.WriteString("\t\tpanic(err)\n")
	source.WriteString("\t}\n")
	source.WriteString("\tif rendered != want {\n")
	source.WriteString("\t\tpanic(fmt.Sprintf(\"%s returned %s, want %s\", name, rendered, want))\n")
	source.WriteString("\t}\n")
	source.WriteString("\tfmt.Println(rendered)\n")
	source.WriteString("}\n")
	return source.String()
}

func moduleGoVersion(path string) string {
	data, err := os.ReadFile(path)
	check(err)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" {
			return fields[1]
		}
	}
	fatalf("no Go version in %s", path)
	return ""
}

func compare(glojureAOT, letGoAOT string, runs int) {
	fmt.Printf("%-18s %-14s %-14s %s\n",
		"workload", "glojure", "let-go", "ratio")
	for _, workload := range workloads {
		glojureOutput, _, err := timedRun(glojureAOT, workload.name)
		checkRun(err, glojureAOT, workload.name, glojureOutput)
		letGoOutput, _, err := timedRun(letGoAOT, workload.name)
		checkRun(err, letGoAOT, workload.name, letGoOutput)
		if !bytes.Equal(glojureOutput, letGoOutput) {
			fatalf(
				"%s produced different output\nglojure: %q\nlet-go:  %q",
				workload.name, glojureOutput, letGoOutput,
			)
		}

		glojureTimes := make([]time.Duration, 0, runs)
		letGoTimes := make([]time.Duration, 0, runs)
		for iteration := 0; iteration < runs; iteration++ {
			if iteration%2 == 0 {
				glojureTimes = append(glojureTimes,
					mustTimedRun(glojureAOT, workload.name))
				letGoTimes = append(letGoTimes,
					mustTimedRun(letGoAOT, workload.name))
			} else {
				letGoTimes = append(letGoTimes,
					mustTimedRun(letGoAOT, workload.name))
				glojureTimes = append(glojureTimes,
					mustTimedRun(glojureAOT, workload.name))
			}
		}

		glojureMedian := median(glojureTimes)
		letGoMedian := median(letGoTimes)
		fmt.Printf("%-18s %-14s %-14s %.3f\n",
			workload.name,
			glojureMedian,
			letGoMedian,
			float64(glojureMedian)/float64(letGoMedian),
		)
	}
}

func mustTimedRun(binary, workload string) time.Duration {
	output, elapsed, err := timedRun(binary, workload)
	checkRun(err, binary, workload, output)
	return elapsed
}

func timedRun(binary, workload string) ([]byte, time.Duration, error) {
	start := time.Now()
	output, err := exec.Command(binary, workload).CombinedOutput()
	return output, time.Since(start), err
}

func checkRun(err error, binary, workload string, output []byte) {
	if err == nil {
		return
	}
	fatalf("%s failed while running %s: %v\n%s",
		binary, workload, err, output)
}

func median(values []time.Duration) time.Duration {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func printSizes(glojureAOT, letGoAOT string) {
	glojureInfo, err := os.Stat(glojureAOT)
	check(err)
	letGoInfo, err := os.Stat(letGoAOT)
	check(err)
	fmt.Printf("\nbinary size: glojure=%s let-go=%s ratio=%.3f\n",
		formatBytes(glojureInfo.Size()),
		formatBytes(letGoInfo.Size()),
		float64(glojureInfo.Size())/float64(letGoInfo.Size()),
	)
}

func formatBytes(size int64) string {
	return fmt.Sprintf("%.2f MiB", float64(size)/(1024*1024))
}

func writeFormattedGo(path, source string) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		panic(fmt.Errorf("format generated %s: %w\n%s", path, err, source))
	}
	writeFile(path, string(formatted))
}

func writeFile(path, contents string) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, []byte(contents), 0o644))
}

func copyFile(source, target string) {
	data, err := os.ReadFile(source)
	check(err)
	check(os.MkdirAll(filepath.Dir(target), 0o755))
	check(os.WriteFile(target, data, 0o644))
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
