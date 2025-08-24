package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/glojurelang/glojure/pkg/codegen"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run clj-to-go.go <clojure-file>")
	}

	inputFile := os.Args[1]
	if !strings.HasSuffix(inputFile, ".clj") {
		log.Fatal("Input file must have .clj extension")
	}

	// Initialize the runtime
	runtime.AddLoadPath(os.DirFS("."))

	// Read and parse the Clojure file
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create reader and parse the namespace
	rdr := reader.New(bufio.NewReader(file), reader.WithFilename(inputFile))

	// Read the first form (should be ns declaration)
	form, err := rdr.ReadOne()
	if err != nil {
		log.Fatal(err)
	}

	// Capture ALL stdout (eval + codegen) so we can prefix it with //
	originalStdout := os.Stdout
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout = stdoutWrite

	// Also bind Clojure's *out* to the same writer so println output is captured
	lang.PushThreadBindings(lang.NewMap(
		lang.VarOut, stdoutWrite,
	))

	capturedLines := make([]string, 0, 16)
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdoutRead)
		for scanner.Scan() {
			capturedLines = append(capturedLines, scanner.Text())
		}
		close(done)
	}()

	// Evaluate the namespace declaration
	env := lang.GlobalEnv
	_, err = env.Eval(form)
	if err != nil {
		// Restore and close before exiting
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		log.Fatal(err)
	}

	// Read and evaluate the rest of the forms
	for {
		val, err := rdr.ReadOne()
		if err == reader.ErrEOF {
			break
		}
		if err != nil {
			stdoutWrite.Close()
			os.Stdout = originalStdout
			lang.PopThreadBindings()
			<-done
			log.Fatal(err)
		}
		_, err = env.Eval(val)
		if err != nil {
			stdoutWrite.Close()
			os.Stdout = originalStdout
			lang.PopThreadBindings()
			<-done
			log.Fatal(err)
		}
	}

	// Extract namespace name from filename
	baseName := strings.TrimSuffix(filepath.Base(inputFile), ".clj")
	nsName := strings.ReplaceAll(baseName, "_", "-")
	nsName = strings.ReplaceAll(nsName, ".", "-")

	// Find the namespace
	ns := lang.FindNamespace(lang.NewSymbol(nsName))
	if ns == nil {
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		log.Fatalf("Namespace %s not found", nsName)
	}

	// Generate Go code to a buffer
	var codegenBuffer bytes.Buffer
	gen := codegen.New(&codegenBuffer)
	if err := gen.Generate(ns); err != nil {
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		log.Fatal(err)
	}

	// Stop capturing stdout and wait for the reader to finish
	stdoutWrite.Close()
	os.Stdout = originalStdout
	lang.PopThreadBindings()
	<-done

	// First, emit any captured stdout lines as comments
	for _, line := range capturedLines {
		fmt.Fprintf(os.Stdout, "// %s\n", line)
	}

	// Then emit the generated code as-is
	scanner := bufio.NewScanner(&codegenBuffer)
	for scanner.Scan() {
		fmt.Fprintln(os.Stdout, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
