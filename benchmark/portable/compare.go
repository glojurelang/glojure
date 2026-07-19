package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"
)

var workloads = []string{
	"prime-workload.clj",
	"event-analytics.clj",
	"game-of-life.clj",
	"mandelbrot.clj",
}

func main() {
	glojure := flag.String("glojure", "./bin/glj", "path to the Glojure executable")
	letGo := flag.String("let-go", "lg", "path to the let-go executable")
	runs := flag.Int("runs", 11, "timed runs per executable and workload")
	dir := flag.String("dir", "benchmark/portable", "directory containing the workloads")
	flag.Parse()

	if *runs < 1 {
		fmt.Fprintln(os.Stderr, "-runs must be at least 1")
		os.Exit(2)
	}

	for _, workload := range workloads {
		path := filepath.Join(*dir, workload)
		glojureOutput, _, err := run(*glojure, path)
		check(err, *glojure, path)
		letGoOutput, _, err := run(*letGo, path)
		check(err, *letGo, path)
		if !bytes.Equal(glojureOutput, letGoOutput) {
			fmt.Fprintf(os.Stderr, "%s produced different output\n", workload)
			fmt.Fprintf(os.Stderr, "Glojure: %q\nlet-go:  %q\n", glojureOutput, letGoOutput)
			os.Exit(1)
		}

		glojureTimes := make([]time.Duration, 0, *runs)
		letGoTimes := make([]time.Duration, 0, *runs)
		for iteration := 0; iteration < *runs; iteration++ {
			if iteration%2 == 0 {
				glojureTimes = append(glojureTimes, timedRun(*glojure, path))
				letGoTimes = append(letGoTimes, timedRun(*letGo, path))
			} else {
				letGoTimes = append(letGoTimes, timedRun(*letGo, path))
				glojureTimes = append(glojureTimes, timedRun(*glojure, path))
			}
		}

		glojureMedian := median(glojureTimes)
		letGoMedian := median(letGoTimes)
		fmt.Printf(
			"%-22s glojure=%-12s let-go=%-12s ratio=%.3f\n",
			workload,
			glojureMedian,
			letGoMedian,
			float64(glojureMedian)/float64(letGoMedian),
		)
	}
}

func timedRun(binary, workload string) time.Duration {
	_, elapsed, err := run(binary, workload)
	check(err, binary, workload)
	return elapsed
}

func run(binary, workload string) ([]byte, time.Duration, error) {
	start := time.Now()
	output, err := exec.Command(binary, workload).CombinedOutput()
	return output, time.Since(start), err
}

func check(err error, binary, workload string) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s failed while running %s: %v\n", binary, workload, err)
	os.Exit(1)
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
