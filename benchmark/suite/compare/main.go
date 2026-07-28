// Command compare builds and compares the portable suite across Glojure
// revisions and alternative native Clojure implementations.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
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
	driver     string
}

type implementation struct {
	name        string
	executable  string
	interpreter bool
}

type sample struct {
	wall time.Duration
	cpu  time.Duration
}

type workloadResult struct {
	workload workload
	cells    []resultCell
}

type resultCell struct {
	samples []sample
	status  string
}

func main() {
	rootFlag := flag.String("root", "", "Glojure repository root (default: auto-detect)")
	mainRefFlag := flag.String("main-ref", "origin/main", "Git ref used for latest main")
	fetchFlag := flag.Bool("fetch", true, "fetch the main ref from origin before building")
	letGoRootFlag := flag.String("let-go-root", "../let-go", "let-go repository root")
	nativeImageFlag := flag.String("native-image", "", "GraalVM native-image executable (default: auto-detect)")
	clojureFlag := flag.String("clojure", "clojure", "Clojure CLI executable used for GraalVM")
	goFlag := flag.String("go", "go", "Go executable")
	runsFlag := flag.Int("runs", 5, "timed process runs per implementation and workload")
	warmupFlag := flag.Int("warmup", 1, "untimed warmups per implementation and workload")
	timeoutFlag := flag.Duration("timeout", 10*time.Minute, "timeout for each benchmark process")
	keepFlag := flag.Bool("keep", false, "retain generated worktrees and executables")
	flag.Parse()

	if *runsFlag < 1 {
		fatalf("-runs must be at least 1")
	}
	if *warmupFlag < 0 {
		fatalf("-warmup must not be negative")
	}

	root := *rootFlag
	if root == "" {
		root = repositoryRoot()
	}
	root = mustAbs(root)
	letGoRoot := mustAbsFrom(root, *letGoRootFlag)
	nativeImage := findNativeImage(root, *nativeImageFlag)
	suite, suiteDir := readManifest(
		filepath.Join(root, "benchmark", "suite", "manifest.json"),
	)

	temp, err := os.MkdirTemp("", "glojure-suite-comparison-")
	check(err)
	if *keepFlag {
		fmt.Fprintf(os.Stderr, "comparison directory: %s\n", temp)
	} else {
		defer os.RemoveAll(temp)
	}

	if *fetchFlag {
		run(root, nil, "git", "fetch", "--quiet", "origin", "main")
	}
	mainCheckout := filepath.Join(temp, "main")
	run(root, nil, "git", "worktree", "add", "--quiet", "--detach",
		mainCheckout, *mainRefFlag)
	if !*keepFlag {
		defer func() {
			runBestEffort(root, "git", "worktree", "remove", "--force", mainCheckout)
		}()
	}

	headCommit := strings.TrimSpace(output(root, nil, "git", "rev-parse", "HEAD"))
	mainCommit := strings.TrimSpace(output(
		mainCheckout, nil, "git", "rev-parse", "HEAD",
	))
	letGoCommit := strings.TrimSpace(output(
		letGoRoot, nil, "git", "rev-parse", "HEAD",
	))

	driverDir := filepath.Join(temp, "drivers")
	check(os.MkdirAll(driverDir, 0o755))
	writeDrivers(suiteDir, driverDir, suite.Workloads)

	fmt.Fprintln(os.Stderr, "building HEAD Glojure interpreter")
	headInterpreter := buildGlojureInterpreter(
		root, filepath.Join(temp, "head-glj"), *goFlag,
	)
	fmt.Fprintln(os.Stderr, "building HEAD Glojure AOT selector")
	headAOT := buildGlojureAOT(
		root,
		suiteDir,
		filepath.Join(temp, "head-aot"),
		suite.Workloads,
		*goFlag,
	)
	fmt.Fprintln(os.Stderr, "building main Glojure interpreter")
	mainInterpreter := buildGlojureInterpreter(
		mainCheckout, filepath.Join(temp, "main-glj"), *goFlag,
	)
	fmt.Fprintln(os.Stderr, "building main Glojure AOT selector")
	mainAOT := buildGlojureAOT(
		mainCheckout,
		suiteDir,
		filepath.Join(temp, "main-aot"),
		suite.Workloads,
		*goFlag,
	)
	fmt.Fprintln(os.Stderr, "building GraalVM selector")
	graalVM := buildGraalVM(
		suiteDir,
		filepath.Join(temp, "graalvm"),
		suite.Workloads,
		*clojureFlag,
		nativeImage,
	)
	fmt.Fprintln(os.Stderr, "building let-go AOT selector")
	letGoAOT := buildLetGoAOT(
		letGoRoot,
		suiteDir,
		filepath.Join(temp, "let-go"),
		suite.Workloads,
		*goFlag,
	)

	implementations := []implementation{
		{name: "HEAD glj interpreter", executable: headInterpreter, interpreter: true},
		{name: "HEAD glj AOT", executable: headAOT},
		{name: "main glj interpreter", executable: mainInterpreter, interpreter: true},
		{name: "main glj AOT", executable: mainAOT},
		{name: "GraalVM", executable: graalVM},
		{name: "let-go AOT", executable: letGoAOT},
	}
	results := compare(
		suite.Workloads,
		implementations,
		*runsFlag,
		*warmupFlag,
		*timeoutFlag,
	)
	printTable(
		results,
		implementations,
		*runsFlag,
		headCommit,
		mainCommit,
		letGoCommit,
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate repository root")
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
	if suite.Version != 1 || len(suite.Workloads) == 0 {
		fatalf("unsupported or empty suite manifest")
	}
	aliases := make(map[string]bool, len(suite.Workloads))
	for i := range suite.Workloads {
		value := &suite.Workloads[i]
		if value.Name == "" || value.Namespace == "" ||
			value.File == "" || value.Expected == "" {
			fatalf("incomplete workload at manifest index %d", i)
		}
		value.alias = goIdentifier(value.Name)
		if aliases[value.alias] {
			fatalf("duplicate generated identifier %q", value.alias)
		}
		aliases[value.alias] = true
		value.importPath = strings.ReplaceAll(
			strings.ReplaceAll(value.Namespace, ".", "/"),
			"-",
			"_",
		)
	}
	return suite, filepath.Dir(path)
}

func writeDrivers(suiteDir, driverDir string, workloads []workload) {
	for i := range workloads {
		source, err := os.ReadFile(filepath.Join(suiteDir, workloads[i].File))
		check(err)
		driver := filepath.Join(driverDir, workloads[i].Name+".clj")
		check(os.WriteFile(
			driver,
			append(source, []byte("\n\n(println (pr-str (run)))\n")...),
			0o644,
		))
		workloads[i].driver = driver
	}
}

func buildGlojureInterpreter(checkout, target, goCommand string) string {
	run(checkout, nil, goCommand, "build", "-o", target, "./cmd/glj")
	return target
}

func buildGlojureAOT(
	checkout string,
	suiteDir string,
	targetDir string,
	workloads []workload,
	goCommand string,
) string {
	check(os.MkdirAll(targetDir, 0o755))
	sourceRoot := filepath.Join(targetDir, "src")
	copyWorkloads(suiteDir, sourceRoot, workloads)
	writeGlojureModule(checkout, targetDir)

	bootstrap := filepath.Join(targetDir, "glj-bootstrap")
	run(checkout, nil, goCommand,
		"build", "-tags", "glj_no_aot_stdlib",
		"-o", bootstrap, "./cmd/glj",
	)
	env := append(os.Environ(),
		"GLJ_CLASSPATH="+sourceRoot,
		"GLOJURE_STDLIB_PATH="+filepath.Join(checkout, "pkg", "stdlib"),
		"GLOJURE_USE_AOT=false",
	)
	for _, value := range workloads {
		run(targetDir, env, bootstrap,
			"-e", "(compile '"+value.Namespace+")")
	}
	writeFormattedGo(
		filepath.Join(targetDir, "main.go"),
		glojureSelectorSource(workloads),
	)
	run(targetDir, nil, goCommand, "mod", "tidy")
	target := filepath.Join(targetDir, "suite-glojure-aot")
	run(targetDir, nil, goCommand,
		"build", "-tags", "glj_aot_runtime",
		"-ldflags", "-s -w", "-o", target, ".",
	)
	return target
}

func writeGlojureModule(checkout, targetDir string) {
	source := fmt.Sprintf(`module glojure-suite-selector

go 1.24

require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => %s
`, filepath.ToSlash(checkout))
	writeFile(filepath.Join(targetDir, "go.mod"), source)
	copyFile(filepath.Join(checkout, "go.sum"), filepath.Join(targetDir, "go.sum"))
}

func glojureSelectorSource(workloads []workload) string {
	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"fmt\"\n\t\"os\"\n\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/glj\"\n")
	source.WriteString("\t\"github.com/glojurelang/glojure/pkg/lang\"\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/stdlib/clojure/core/protocols\"\n")
	source.WriteString("\t_ \"github.com/glojurelang/glojure/pkg/stdlib/clojure/string\"\n")
	for _, value := range workloads {
		fmt.Fprintf(
			&source,
			"\t%s \"glojure-suite-selector/src/%s\"\n",
			value.alias,
			value.importPath,
		)
	}
	source.WriteString(")\n\n")
	source.WriteString("type target struct {\n")
	source.WriteString("\tload func()\n\tnamespace string\n\texpected string\n")
	source.WriteString("}\n\n")
	source.WriteString("var targets = map[string]target{\n")
	for _, value := range workloads {
		fmt.Fprintf(
			&source,
			"\t%q: {%s.LoadNS, %q, %q},\n",
			value.Name,
			value.alias,
			value.Namespace,
			value.Expected,
		)
	}
	source.WriteString("}\n\n")
	source.WriteString("func main() {\n")
	source.WriteString("\tif len(os.Args) != 2 { panic(\"expected workload name\") }\n")
	source.WriteString("\ttarget, ok := targets[os.Args[1]]\n")
	source.WriteString("\tif !ok { panic(\"unknown workload\") }\n")
	source.WriteString("\ttarget.load()\n")
	source.WriteString("\tns := lang.FindNamespace(lang.NewSymbol(target.namespace))\n")
	source.WriteString("\trun := ns.FindInternedVar(lang.NewSymbol(\"run\")).Get().(lang.IFn)\n")
	source.WriteString("\tresult := lang.PrintString(run.Invoke())\n")
	source.WriteString("\tif result != target.expected {\n")
	source.WriteString("\t\tpanic(fmt.Sprintf(\"got %s, want %s\", result, target.expected))\n")
	source.WriteString("\t}\n")
	source.WriteString("\tfmt.Println(result)\n")
	source.WriteString("}\n")
	return source.String()
}

func buildGraalVM(
	suiteDir string,
	targetDir string,
	workloads []workload,
	clojureCommand string,
	nativeImage string,
) string {
	sourceRoot := filepath.Join(targetDir, "src")
	classRoot := filepath.Join(targetDir, "classes")
	check(os.MkdirAll(classRoot, 0o755))
	copyWorkloads(suiteDir, sourceRoot, workloads)
	driverPath := filepath.Join(
		sourceRoot, "benchmark", "suite", "driver.clj",
	)
	writeFile(driverPath, graalDriverSource(workloads))

	sdeps := fmt.Sprintf(
		`{:paths [%q %q]}`,
		filepath.ToSlash(sourceRoot),
		filepath.ToSlash(classRoot),
	)
	compileExpression := fmt.Sprintf(
		`(binding [*compile-path* %q] (compile 'benchmark.suite.driver))`,
		filepath.ToSlash(classRoot),
	)
	run(targetDir, nil, clojureCommand,
		"-J-Dclojure.compiler.direct-linking=true",
		"-Sdeps", sdeps,
		"-M", "-e", compileExpression,
	)
	classpath := strings.TrimSpace(output(
		targetDir,
		nil,
		clojureCommand,
		"-Sdeps", sdeps,
		"-Spath",
	))
	target := filepath.Join(targetDir, "suite-graalvm")
	run(targetDir, nil, nativeImage,
		"-O1",
		"--no-fallback",
		"--initialize-at-build-time",
		"--initialize-at-run-time=clojure.lang.Compiler",
		"-H:+ReportExceptionStackTraces",
		"-J-Dclojure.compiler.direct-linking=true",
		"-J-Xmx4g",
		"-cp", classpath,
		"-o", target,
		"benchmark.suite.driver",
	)
	return target
}

func graalDriverSource(workloads []workload) string {
	var source strings.Builder
	source.WriteString("(ns benchmark.suite.driver\n")
	source.WriteString("  (:gen-class)\n  (:require\n")
	for _, value := range workloads {
		fmt.Fprintf(&source, "    [%s]\n", value.Namespace)
	}
	source.WriteString("    ))\n\n")
	source.WriteString("(def targets\n  {")
	for _, value := range workloads {
		fmt.Fprintf(
			&source,
			"\n   %q [(var %s/run) %q]",
			value.Name,
			value.Namespace,
			value.Expected,
		)
	}
	source.WriteString("})\n\n")
	source.WriteString("(defn -main [& args]\n")
	source.WriteString("  (let [[run expected] (get targets (first args))]\n")
	source.WriteString("    (when (nil? run) (throw (ex-info \"unknown workload\" {:args args})))\n")
	source.WriteString("    (let [result (pr-str (run))]\n")
	source.WriteString("      (when (not= result expected)\n")
	source.WriteString("        (throw (ex-info \"wrong result\" {:got result :expected expected})))\n")
	source.WriteString("      (println result))))\n")
	return source.String()
}

func buildLetGoAOT(
	letGoRoot string,
	suiteDir string,
	targetDir string,
	workloads []workload,
	goCommand string,
) string {
	clone := filepath.Join(targetDir, "let-go")
	check(os.MkdirAll(targetDir, 0o755))
	run(targetDir, nil, "git", "clone", "--quiet", "--local",
		"--no-hardlinks", letGoRoot, clone)

	compiler := filepath.Join(clone, "lg")
	run(clone, nil, goCommand, "build", "-o", compiler, ".")
	run(clone, nil, goCommand,
		"run", "-tags", "bootstrap", "./cmd/lgbgen", "--target=go",
	)

	module := filepath.Join(targetDir, "selector")
	generated := filepath.Join(module, "generated")
	args := []string{
		"scripts/lg-compile",
		generated,
		"glojure-suite-letgo/generated",
	}
	for _, value := range workloads {
		args = append(args, filepath.Join(suiteDir, value.File))
	}
	run(clone, nil, compiler, args...)

	goVersion := moduleGoVersion(filepath.Join(clone, "go.mod"))
	writeFile(filepath.Join(module, "go.mod"), fmt.Sprintf(`module glojure-suite-letgo

go %s

require github.com/nooga/let-go v0.0.0

replace github.com/nooga/let-go => %s
`, goVersion, filepath.ToSlash(clone)))
	copyFile(filepath.Join(clone, "go.sum"), filepath.Join(module, "go.sum"))
	copyFile(
		filepath.Join(clone, "lg_gogen_ir.go"),
		filepath.Join(module, "core_gogen_ir.go"),
	)
	writeFormattedGo(
		filepath.Join(module, "main.go"),
		letGoSelectorSource(workloads),
	)
	run(module, nil, goCommand, "mod", "tidy")
	target := filepath.Join(targetDir, "suite-let-go-aot")
	run(module, nil, goCommand,
		"build", "-tags", "gogen_ir",
		"-ldflags", "-s -w", "-o", target, ".",
	)
	return target
}

func letGoSelectorSource(workloads []workload) string {
	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("import (\n\t\"fmt\"\n\t\"os\"\n\n")
	source.WriteString("\t\"github.com/nooga/let-go/pkg/rt\"\n")
	source.WriteString("\t_ \"github.com/nooga/let-go/pkg/rt/corefns\"\n")
	source.WriteString("\t\"github.com/nooga/let-go/pkg/vm\"\n")
	for _, value := range workloads {
		fmt.Fprintf(
			&source,
			"\t_ \"glojure-suite-letgo/generated/%s\"\n",
			value.importPath,
		)
	}
	source.WriteString(")\n\n")
	source.WriteString("type target struct { namespace, expected string }\n")
	source.WriteString("var targets = map[string]target{\n")
	for _, value := range workloads {
		fmt.Fprintf(
			&source,
			"\t%q: {%q, %q},\n",
			value.Name,
			value.Namespace,
			value.Expected,
		)
	}
	source.WriteString("}\n\n")
	source.WriteString("func main() {\n")
	source.WriteString("\tif len(os.Args) != 2 { panic(\"expected workload name\") }\n")
	source.WriteString("\ttarget, ok := targets[os.Args[1]]\n")
	source.WriteString("\tif !ok { panic(\"unknown workload\") }\n")
	source.WriteString("\tec, err := rt.BootCore()\n")
	source.WriteString("\tif err != nil { panic(err) }\n")
	source.WriteString("\tns := rt.LookupOrRegisterNS(target.namespace)\n")
	source.WriteString("\trt.ApplyGoOverrides(ns)\n")
	source.WriteString("\trunVar, ok := ns.Lookup(vm.Symbol(\"run\")).(*vm.Var)\n")
	source.WriteString("\tif !ok { panic(\"missing run Var\") }\n")
	source.WriteString("\trun, ok := runVar.Deref().(*vm.NativeFn)\n")
	source.WriteString("\tif !ok { panic(fmt.Sprintf(\"run is not AOT: %T\", runVar.Deref())) }\n")
	source.WriteString("\tresult, err := ec.Invoke(run, nil)\n")
	source.WriteString("\tif err != nil { panic(err) }\n")
	source.WriteString("\trendered, err := vm.SafeString(result)\n")
	source.WriteString("\tif err != nil { panic(err) }\n")
	source.WriteString("\tif rendered != target.expected {\n")
	source.WriteString("\t\tpanic(fmt.Sprintf(\"got %s, want %s\", rendered, target.expected))\n")
	source.WriteString("\t}\n")
	source.WriteString("\tfmt.Println(rendered)\n")
	source.WriteString("}\n")
	return source.String()
}

func copyWorkloads(suiteDir, sourceRoot string, workloads []workload) {
	for _, value := range workloads {
		source := filepath.Join(suiteDir, value.File)
		target := filepath.Join(sourceRoot, value.importPath+".clj")
		copyFile(source, target)
	}
}

func compare(
	workloads []workload,
	implementations []implementation,
	runs int,
	warmup int,
	timeout time.Duration,
) []workloadResult {
	results := make([]workloadResult, 0, len(workloads))
	for _, value := range workloads {
		fmt.Fprintf(os.Stderr, "benchmarking %s\n", value.Name)
		result := workloadResult{
			workload: value,
			cells:    make([]resultCell, len(implementations)),
		}
		for i, implementation := range implementations {
			rendered, _, err := invoke(timeout, implementation, value)
			if err != nil {
				result.cells[i].status = "ERROR"
				fmt.Fprintf(
					os.Stderr,
					"%s failed for %s: %v\n%s\n",
					implementation.name,
					value.Name,
					err,
					bytes.TrimSpace(rendered),
				)
				continue
			}
			if actual := strings.TrimSpace(string(rendered)); actual != value.Expected {
				result.cells[i].status = "WRONG RESULT"
				fmt.Fprintf(
					os.Stderr,
					"%s returned %q for %s, want %q\n",
					implementation.name,
					actual,
					value.Name,
					value.Expected,
				)
				continue
			}
			warmupFailed := false
			for range warmup {
				rendered, _, err := invoke(timeout, implementation, value)
				if err != nil {
					result.cells[i].status = "WARMUP ERROR"
					fmt.Fprintf(
						os.Stderr,
						"%s warmup failed for %s: %v\n%s\n",
						implementation.name,
						value.Name,
						err,
						bytes.TrimSpace(rendered),
					)
					warmupFailed = true
					break
				}
			}
			if warmupFailed {
				continue
			}
			result.cells[i].samples = make([]sample, 0, runs)
		}
		for iteration := range runs {
			for offset := range implementations {
				index := (iteration + offset) % len(implementations)
				if result.cells[index].status != "" {
					continue
				}
				rendered, measured, err := invoke(
					timeout, implementations[index], value,
				)
				if err != nil {
					result.cells[index].status = "ERROR"
					result.cells[index].samples = nil
					fmt.Fprintf(os.Stderr, "%s failed for %s: %v\n%s\n",
						implementations[index].name,
						value.Name,
						err,
						bytes.TrimSpace(rendered),
					)
					continue
				}
				result.cells[index].samples = append(
					result.cells[index].samples, measured,
				)
			}
		}
		results = append(results, result)
	}
	return results
}

func invoke(
	timeout time.Duration,
	implementation implementation,
	workload workload,
) ([]byte, sample, error) {
	argument := workload.Name
	if implementation.interpreter {
		argument = workload.driver
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, implementation.executable, argument)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	measured := sample{wall: time.Since(start)}
	if cmd.ProcessState != nil {
		measured.cpu = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	}
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		return append(stdout.Bytes(), stderr.Bytes()...), measured, err
	}
	return stdout.Bytes(), measured, nil
}

func printTable(
	results []workloadResult,
	implementations []implementation,
	runs int,
	headCommit string,
	mainCommit string,
	letGoCommit string,
) {
	fmt.Printf(
		"Comparison uses %d timed process runs per cell; values are median wall ms (CPU ms).\n\n",
		runs,
	)
	fmt.Printf(
		"HEAD `%s`; main `%s`; let-go `%s`.\n\n",
		shortCommit(headCommit),
		shortCommit(mainCommit),
		shortCommit(letGoCommit),
	)
	fmt.Print("| workload |")
	for _, implementation := range implementations {
		fmt.Printf(" %s |", implementation.name)
	}
	fmt.Print("\n| --- |")
	for range implementations {
		fmt.Print(" ---: |")
	}
	fmt.Println()
	for _, result := range results {
		fmt.Printf("| %s |", result.workload.Name)
		for _, cell := range result.cells {
			if cell.status != "" {
				fmt.Printf(" **%s** |", cell.status)
			} else {
				fmt.Printf(" %s |", formatSamples(cell.samples))
			}
		}
		fmt.Println()
	}
}

func formatSamples(samples []sample) string {
	walls := make([]time.Duration, len(samples))
	cpus := make([]time.Duration, len(samples))
	for i, value := range samples {
		walls[i] = value.wall
		cpus[i] = value.cpu
	}
	return fmt.Sprintf(
		"%.3f (%.3f)",
		float64(median(walls))/float64(time.Millisecond),
		float64(median(cpus))/float64(time.Millisecond),
	)
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

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func findNativeImage(root, configured string) string {
	candidates := []string{
		configured,
		os.Getenv("NATIVE_IMAGE"),
	}
	if found, err := exec.LookPath("native-image"); err == nil {
		candidates = append(candidates, found)
	}
	candidates = append(candidates, filepath.Join(
		filepath.Dir(root),
		"yamlstar",
		".cache",
		"local",
		"graalvm-jdk-25",
		"Contents",
		"Home",
		"lib",
		"svm",
		"bin",
		"native-image",
	))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		absolute := mustAbsFrom(root, candidate)
		if info, err := os.Stat(absolute); err == nil &&
			!info.IsDir() && info.Mode()&0o111 != 0 {
			return absolute
		}
	}
	fatalf("cannot find GraalVM native-image; use -native-image")
	return ""
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

func writeFormattedGo(path, source string) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		fatalf("format generated %s: %v\n%s", path, err, source)
	}
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, formatted, 0o644))
}

func writeFile(path, source string) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, []byte(source), 0o644))
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

func mustAbsFrom(root, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return mustAbs(path)
}

func output(dir string, env []string, command string, args ...string) string {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fatalf("%s failed: %v\n%s",
			strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return stdout.String()
}

func run(dir string, env []string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "+ %s\n", strings.Join(cmd.Args, " "))
	check(cmd.Run())
}

func runBestEffort(dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cleanup failed: %v\n", err)
	}
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
