package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"

	// Bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: glj-aot <clojure-file>")
	}

	inputFile := os.Args[1]

	// Read and parse the Clojure file
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create reader and parse the namespace
	rdr := reader.New(bufio.NewReader(file), reader.WithFilename(inputFile))

	// Read the first form
	form, err := rdr.ReadOne()
	if err != nil {
		log.Fatal(err)
	}

	// Validate that the first form is a namespace declaration
	if list, ok := form.(lang.ISeq); ok {
		first := lang.First(list)
		if sym, ok := first.(*lang.Symbol); ok && sym.Name() != "ns" {
			log.Fatal("First form must be a namespace declaration (ns)")
		}
	} else {
		log.Fatal("First form must be a namespace declaration (ns)")
	}

	// Capture and discard any output produced during AOT compilation, preventing
	// it from being included with the generated Go code:
	originalStdout := os.Stdout
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout = stdoutWrite

	lang.PushThreadBindings(lang.NewMap(
		lang.VarOut, stdoutWrite,
	))

	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdoutRead)
		for scanner.Scan() {}
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

	// Get the namespace from the environment after evaluating the ns form
	ns := env.CurrentNamespace()
	if ns == nil {
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		log.Fatalf("Failed to get current namespace")
	}

	// Generate Go code to a buffer using the new LoadNS techniques
	var codegenBuffer bytes.Buffer
	gen := runtime.NewGenerator(&codegenBuffer)
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

	// Transform the generated code to main package format only if namespace is 'main'
	var outputCode []byte
	if ns.Name().Name() == "main" {
		outputCode = transformToMainPackage(codegenBuffer.Bytes())
	} else {
		outputCode = codegenBuffer.Bytes()
	}

	// Emit the transformed code
	os.Stdout.Write(outputCode)
}

// transformToMainPackage transforms the generated code to a main package
// with command-line argument handling
func transformToMainPackage(generatedCode []byte) []byte {
	code := string(generatedCode)

	// Change package from "generated" to "main"
	code = strings.Replace(code, "package generated", "package main", 1)

	// Add extra imports
	code = strings.Replace(
		code,
		"\nimport (",
		`
import (
	_ "github.com/glojurelang/glojure/pkg/glj"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core"
	os "os"`,
		1)

	// Add callMainFunc at the end of the init function
	lines := strings.Split(code, "\n")
	initFuncEnd := 0
	inInitFunc := false

	for i, line := range lines {
		if line == "func init() {" {
			inInitFunc = true
		}

		if inInitFunc {
			if line == "}" {
				initFuncEnd = i
				break
			}
		}
	}

	if initFuncEnd == 0 {
		log.Fatal("Failed to find the end of the init function")
	}

	// Insert the code before the closing brace of the init function
	code = strings.Join(lines[:initFuncEnd], "\n") + `
	callMainFunc()
` + strings.Join(lines[initFuncEnd:], "\n")

	// Add the callMainFunc function at the end
	code = code + `

// Call the Glojure main function with command-line arguments
func callMainFunc() {
	LoadNS()

	args := os.Args[1:]
	argsAny := make([]any, len(args))
	for i, s := range args { argsAny[i] = s }

	// YS: Find and call the 'main' function
	ns := lang.FindNamespace(lang.NewSymbol("main"))
	mainVar := ns.FindInternedVar(lang.NewSymbol("main"))
	if mainVar != nil && !mainVar.IsMacro() {
		mainFn := mainVar.Get()
		lang.Apply(mainFn, argsAny)
	}
}

// Dummy main function to compile to a standalone executable
func main() {}
`

	return []byte(code)
}
