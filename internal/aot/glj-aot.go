package main

import (
	"bufio"
	"bytes"
	_ "fmt"
	"log"
	"os"
	"strings"

	"github.com/glojurelang/glojure/pkg/codegen"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"

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
	code = strings.ReplaceAll(code, "package generated", "package main")

	// Add imports for os and glj package
	code = strings.ReplaceAll(code,
		"\nimport (",
		`
import (
	os "os"
	_ "github.com/glojurelang/glojure/pkg/glj"`)

	// Add empty main function after the imports
	// Look for the pattern where imports end and func init() begins
	code = strings.ReplaceAll(code,
		`)

func init()`,
		`)

func main() {}

func init()`)

	// Add command-line argument handling at the end of init function
	// Find the end of the init function and insert the code
	lines := strings.Split(code, "\n")
	var initFuncEnd int

	// Look for the end of the init function by finding the last closing brace
	// that belongs to the init function
	// We need to find the outermost closing brace of the init function
	braceCount := 0
	inInitFunc := false

	for i, line := range lines {
		if strings.Contains(line, "func init()") {
			inInitFunc = true
			braceCount = 0
		}

		if inInitFunc {
			if strings.Contains(line, "{") {
				braceCount++
			}
			if strings.Contains(line, "}") {
				braceCount--
				if braceCount == 0 {
					// This is the end of the init function
					initFuncEnd = i
					break
				}
			}
		}
	}

	if initFuncEnd > 0 {
		// Insert command-line argument handling before the closing brace
		cmdLineCode := `
		// Command-line argument handling
		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		args := os.Args[1:]
		cla := lang.NewSymbol("*command-line-args*")
		core.FindInternedVar(cla).BindRoot(lang.Seq(args))
		argsAny := make([]any, len(args))
		for i, s := range args { argsAny[i] = s }

		// Find and call the main function
		mainVar := ns.FindInternedVar(lang.NewSymbol("main"))
		if mainVar != nil && !mainVar.IsMacro() {
			mainFn := mainVar.Get()
			lang.Apply(mainFn, argsAny)
		}
`

		// Insert the code before the closing brace
		beforeBrace := strings.Join(lines[:initFuncEnd], "\n")
		afterBrace := strings.Join(lines[initFuncEnd:], "\n")
		code = beforeBrace + cmdLineCode + afterBrace
	}

	return []byte(code)
}
