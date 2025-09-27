package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/gljmain"
)

func main() {
	// Command-line flags
	var (
		dirs       = flag.String("dir", "test", "Comma-separated test directories")
		namespaces = flag.String("namespace", "", "Namespace pattern (regex)")
		format     = flag.String("format", "console", "Output format: console, tap, json, junit, edn")
		outputFile = flag.String("output", "", "Output file (stdout if not specified)")
		failFast   = flag.Bool("fail-fast", false, "Stop on first failure")
		includes   = flag.String("include", "", "Include tests with metadata (comma-separated)")
		excludes   = flag.String("exclude", "", "Exclude tests with metadata (comma-separated)")
		parallel   = flag.Int("parallel", 1, "Number of parallel test runners")
		verbose    = flag.Bool("verbose", false, "Verbose output")
		listTests  = flag.Bool("list", false, "List test namespaces without running")
	)

	flag.Parse()

	// Build the Clojure code to run the test runner
	args := buildTestRunnerArgs(*dirs, *namespaces, *format, *outputFile, 
		*failFast, *includes, *excludes, *parallel, *verbose, *listTests)
	
	// Create the test runner invocation
	runnerCode := fmt.Sprintf(`
		(require 'glojure.test-runner)
		(glojure.test-runner/run-tests %s)
	`, args)

	// Write the code to a temporary file and run it
	// This is more reliable than using -e with complex code
	tmpFile, err := os.CreateTemp("", "test-runner-*.glj")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(runnerCode); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to temp file: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()
	
	// Run the test runner script
	gljArgs := []string{tmpFile.Name()}
	gljmain.Main(gljArgs)
}

func buildTestRunnerArgs(dirs, namespaces, format, outputFile string, 
	failFast bool, includes, excludes string, parallel int, verbose, listTests bool) string {
	
	var opts []string
	
	// Add directories
	if dirs != "" {
		dirList := strings.Split(dirs, ",")
		quotedDirs := make([]string, len(dirList))
		for i, d := range dirList {
			quotedDirs[i] = fmt.Sprintf(`"%s"`, strings.TrimSpace(d))
		}
		opts = append(opts, fmt.Sprintf(":dirs [%s]", strings.Join(quotedDirs, " ")))
	}
	
	// Add namespace pattern
	if namespaces != "" {
		opts = append(opts, fmt.Sprintf(`:namespace-pattern #"%s"`, namespaces))
	}
	
	// Add format
	opts = append(opts, fmt.Sprintf(":format :%s", format))
	
	// Add output file
	if outputFile != "" {
		opts = append(opts, fmt.Sprintf(`:output "%s"`, outputFile))
	}
	
	// Add fail-fast
	if failFast {
		opts = append(opts, ":fail-fast true")
	}
	
	// Add includes
	if includes != "" {
		incList := strings.Split(includes, ",")
		keywords := make([]string, len(incList))
		for i, inc := range incList {
			keywords[i] = ":" + strings.TrimSpace(inc)
		}
		opts = append(opts, fmt.Sprintf(":includes [%s]", strings.Join(keywords, " ")))
	}
	
	// Add excludes
	if excludes != "" {
		excList := strings.Split(excludes, ",")
		keywords := make([]string, len(excList))
		for i, exc := range excList {
			keywords[i] = ":" + strings.TrimSpace(exc)
		}
		opts = append(opts, fmt.Sprintf(":excludes [%s]", strings.Join(keywords, " ")))
	}
	
	// Add parallel
	if parallel > 1 {
		opts = append(opts, fmt.Sprintf(":parallel %d", parallel))
	}
	
	// Add verbose
	if verbose {
		opts = append(opts, ":verbose true")
	}
	
	// Add list-only
	if listTests {
		opts = append(opts, ":list-only true")
	}
	
	return fmt.Sprintf("{%s}", strings.Join(opts, " "))
}