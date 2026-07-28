package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	File      string   `json:"file"`
	Expected  string   `json:"expected"`
	Tags      []string `json:"tags"`
}

type runtimeSpec struct {
	Name    string
	Command []string
}

type runtimeFlags []runtimeSpec

func (f *runtimeFlags) String() string {
	parts := make([]string, len(*f))
	for i, runtime := range *f {
		parts[i] = runtime.Name + "=" + strings.Join(runtime.Command, ",")
	}
	return strings.Join(parts, " ")
}

func (f *runtimeFlags) Set(value string) error {
	runtime, err := parseRuntime(value)
	if err != nil {
		return err
	}
	for _, existing := range *f {
		if existing.Name == runtime.Name {
			return fmt.Errorf("runtime %q was specified more than once", runtime.Name)
		}
	}
	*f = append(*f, runtime)
	return nil
}

type sample struct {
	Wall time.Duration
	User time.Duration
	Sys  time.Duration
}

type jsonSample struct {
	WallNS int64 `json:"wall_ns"`
	UserNS int64 `json:"user_ns"`
	SysNS  int64 `json:"sys_ns"`
}

type jsonRuntimeResult struct {
	Name    string       `json:"name"`
	Samples []jsonSample `json:"samples"`
}

type jsonWorkloadResult struct {
	Name     string              `json:"name"`
	Runtimes []jsonRuntimeResult `json:"runtimes"`
}

type jsonReport struct {
	ManifestVersion int                  `json:"manifest_version"`
	Runs            int                  `json:"runs"`
	Warmup          int                  `json:"warmup"`
	Results         []jsonWorkloadResult `json:"results"`
}

func main() {
	os.Exit(runMain(os.Args[1:]))
}

func runMain(args []string) int {
	flags := flag.NewFlagSet("glojure-benchmark-suite", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	defaultManifest := filepath.Join(suiteRoot(), "manifest.json")
	manifestPath := flags.String("manifest", defaultManifest, "path to suite manifest")
	runs := flags.Int("runs", 7, "timed samples per runtime and workload")
	warmup := flags.Int("warmup", 1, "untimed warmup runs per runtime and workload")
	workloadPattern := flags.String("workload", ".", "regular expression selecting workload names or tags")
	format := flags.String("format", "text", "output format: text or json")
	timeout := flags.Duration("timeout", 5*time.Minute, "timeout for each process")
	list := flags.Bool("list", false, "list selected workloads without running them")
	var runtimes runtimeFlags
	flags.Var(&runtimes, "runtime", "runtime as NAME=EXECUTABLE[,ARG...] (repeatable)")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return 2
	}
	if *runs < 1 {
		fmt.Fprintln(os.Stderr, "-runs must be at least 1")
		return 2
	}
	if *warmup < 0 {
		fmt.Fprintln(os.Stderr, "-warmup must not be negative")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(os.Stderr, "-format must be text or json")
		return 2
	}

	suite, manifestDir, err := readManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "manifest: %v\n", err)
		return 1
	}
	pattern, err := regexp.Compile(*workloadPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "-workload: %v\n", err)
		return 2
	}
	workloads := selectWorkloads(suite.Workloads, pattern)
	if len(workloads) == 0 {
		fmt.Fprintln(os.Stderr, "no workloads matched")
		return 1
	}
	if *list {
		for _, workload := range workloads {
			fmt.Printf("%-20s %s\n", workload.Name, strings.Join(workload.Tags, ","))
		}
		return 0
	}
	if len(runtimes) == 0 {
		fmt.Fprintln(os.Stderr, "at least one -runtime is required")
		return 2
	}

	report, err := benchmarkSuite(
		suite.Version,
		manifestDir,
		workloads,
		runtimes,
		*runs,
		*warmup,
		*timeout,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
			return 1
		}
		return 0
	}
	printReport(report)
	return 0
}

func suiteRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate benchmark suite")
	}
	return filepath.Dir(file)
}

func parseRuntime(value string) (runtimeSpec, error) {
	name, commandText, found := strings.Cut(value, "=")
	if !found || name == "" || commandText == "" {
		return runtimeSpec{}, errors.New("runtime must be NAME=EXECUTABLE[,ARG...]")
	}
	reader := csv.NewReader(strings.NewReader(commandText))
	reader.TrimLeadingSpace = true
	command, err := reader.Read()
	if err != nil {
		return runtimeSpec{}, fmt.Errorf("runtime %q command: %w", name, err)
	}
	if len(command) == 0 || command[0] == "" {
		return runtimeSpec{}, fmt.Errorf("runtime %q has no executable", name)
	}
	return runtimeSpec{Name: name, Command: command}, nil
}

func readManifest(path string) (manifest, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return manifest{}, "", err
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return manifest{}, "", err
	}
	var suite manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&suite); err != nil {
		return manifest{}, "", err
	}
	dir := filepath.Dir(absolute)
	if err := validateManifest(suite, dir); err != nil {
		return manifest{}, "", err
	}
	return suite, dir, nil
}

func validateManifest(suite manifest, dir string) error {
	if suite.Version != 1 {
		return fmt.Errorf("unsupported version %d", suite.Version)
	}
	if len(suite.Workloads) == 0 {
		return errors.New("no workloads")
	}
	names := make(map[string]bool, len(suite.Workloads))
	namespaces := make(map[string]bool, len(suite.Workloads))
	for i, workload := range suite.Workloads {
		label := fmt.Sprintf("workload %d", i)
		if workload.Name == "" {
			return fmt.Errorf("%s has no name", label)
		}
		label = fmt.Sprintf("workload %q", workload.Name)
		if names[workload.Name] {
			return fmt.Errorf("duplicate %s", label)
		}
		names[workload.Name] = true
		if workload.Namespace == "" {
			return fmt.Errorf("%s has no namespace", label)
		}
		if namespaces[workload.Namespace] {
			return fmt.Errorf("duplicate namespace %q", workload.Namespace)
		}
		namespaces[workload.Namespace] = true
		if workload.Expected == "" {
			return fmt.Errorf("%s has no expected output", label)
		}
		if workload.File == "" || filepath.IsAbs(workload.File) {
			return fmt.Errorf("%s has invalid file %q", label, workload.File)
		}
		absolute := filepath.Clean(filepath.Join(dir, workload.File))
		relative, err := filepath.Rel(dir, absolute)
		if err != nil || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%s file escapes manifest directory", label)
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return fmt.Errorf("%s file: %w", label, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s file is not regular", label)
		}
	}
	return nil
}

func selectWorkloads(workloads []workload, pattern *regexp.Regexp) []workload {
	selected := make([]workload, 0, len(workloads))
	for _, workload := range workloads {
		searchable := workload.Name + " " + strings.Join(workload.Tags, " ")
		if pattern.MatchString(searchable) {
			selected = append(selected, workload)
		}
	}
	return selected
}

func benchmarkSuite(
	version int,
	manifestDir string,
	workloads []workload,
	runtimes []runtimeSpec,
	runs int,
	warmup int,
	timeout time.Duration,
) (jsonReport, error) {
	report := jsonReport{
		ManifestVersion: version,
		Runs:            runs,
		Warmup:          warmup,
	}
	driverDir, err := os.MkdirTemp("", "glojure-benchmark-drivers-")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(driverDir)

	for _, workload := range workloads {
		source := filepath.Join(manifestDir, workload.File)
		sourceData, err := os.ReadFile(source)
		if err != nil {
			return report, err
		}
		driver := filepath.Join(driverDir, workload.Name+".clj")
		if err := os.WriteFile(
			driver,
			[]byte(driverSource(sourceData)),
			0o644,
		); err != nil {
			return report, err
		}

		result := jsonWorkloadResult{Name: workload.Name}
		samples := make([][]sample, len(runtimes))
		for runtimeIndex, runtime := range runtimes {
			output, _, err := invoke(timeout, runtime, driver)
			if err != nil {
				return report, invocationError(runtime, workload, output, err)
			}
			if actual := strings.TrimSpace(string(output)); actual != workload.Expected {
				return report, fmt.Errorf(
					"%s produced the wrong result for %s: got %q, want %q",
					runtime.Name,
					workload.Name,
					actual,
					workload.Expected,
				)
			}
			for i := 0; i < warmup; i++ {
				output, _, err := invoke(timeout, runtime, driver)
				if err != nil {
					return report, invocationError(runtime, workload, output, err)
				}
			}
			samples[runtimeIndex] = make([]sample, 0, runs)
		}
		for iteration := 0; iteration < runs; iteration++ {
			for offset := range runtimes {
				runtimeIndex := (iteration + offset) % len(runtimes)
				runtime := runtimes[runtimeIndex]
				output, measured, err := invoke(timeout, runtime, driver)
				if err != nil {
					return report, invocationError(runtime, workload, output, err)
				}
				samples[runtimeIndex] = append(samples[runtimeIndex], measured)
			}
		}
		for runtimeIndex, runtime := range runtimes {
			runtimeResult := jsonRuntimeResult{Name: runtime.Name}
			for _, sample := range samples[runtimeIndex] {
				runtimeResult.Samples = append(runtimeResult.Samples, jsonSample{
					WallNS: int64(sample.Wall),
					UserNS: int64(sample.User),
					SysNS:  int64(sample.Sys),
				})
			}
			result.Runtimes = append(result.Runtimes, runtimeResult)
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func driverSource(source []byte) string {
	return string(source) + "\n\n(println (pr-str (run)))\n"
}

func invoke(
	timeout time.Duration,
	runtime runtimeSpec,
	driver string,
) ([]byte, sample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := append(slices.Clone(runtime.Command[1:]), driver)
	cmd := exec.CommandContext(ctx, runtime.Command[0], args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	measured := sample{Wall: time.Since(start)}
	if cmd.ProcessState != nil {
		measured.User = cmd.ProcessState.UserTime()
		measured.Sys = cmd.ProcessState.SystemTime()
	}
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		output := append(slices.Clone(stdout.Bytes()), stderr.Bytes()...)
		return output, measured, err
	}
	return stdout.Bytes(), measured, nil
}

func invocationError(
	runtime runtimeSpec,
	workload workload,
	output []byte,
	err error,
) error {
	return fmt.Errorf(
		"%s failed for %s: %w\n%s",
		runtime.Name,
		workload.Name,
		err,
		bytes.TrimSpace(output),
	)
}

func printReport(report jsonReport) {
	fmt.Printf(
		"%-20s %-14s %12s %12s %12s %12s %8s\n",
		"workload",
		"runtime",
		"wall median",
		"user median",
		"sys median",
		"wall p95",
		"ratio",
	)
	for _, workload := range report.Results {
		var baseline time.Duration
		for runtimeIndex, runtime := range workload.Runtimes {
			wall := sampleDurations(runtime.Samples, func(sample jsonSample) int64 {
				return sample.WallNS
			})
			user := sampleDurations(runtime.Samples, func(sample jsonSample) int64 {
				return sample.UserNS
			})
			sys := sampleDurations(runtime.Samples, func(sample jsonSample) int64 {
				return sample.SysNS
			})
			wallMedian := median(wall)
			if runtimeIndex == 0 {
				baseline = wallMedian
			}
			fmt.Printf(
				"%-20s %-14s %12s %12s %12s %12s %8.3f\n",
				workload.Name,
				runtime.Name,
				wallMedian,
				median(user),
				median(sys),
				percentile(wall, 0.95),
				float64(wallMedian)/float64(baseline),
			)
		}
	}
}

func sampleDurations(
	samples []jsonSample,
	value func(jsonSample) int64,
) []time.Duration {
	durations := make([]time.Duration, len(samples))
	for i, sample := range samples {
		durations[i] = time.Duration(value(sample))
	}
	return durations
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

func percentile(values []time.Duration, quantile float64) time.Duration {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	index := int(float64(len(sorted)-1)*quantile + 0.5)
	return sorted[index]
}
