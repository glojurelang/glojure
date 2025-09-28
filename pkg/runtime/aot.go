package runtime

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
)

// Compile Clojure source code to Go code using AOT compilation
func CompileAOTString(source string, filename string) error {
	rdr := reader.New(strings.NewReader(source), reader.WithFilename(filename))

	// Capture and discard any output produced during AOT compilation
	originalStdout := os.Stdout
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return err
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

	// Evaluate all forms in the source
	env := lang.GlobalEnv
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
			return err
		}
		_, err = env.Eval(val)
		if err != nil {
			stdoutWrite.Close()
			os.Stdout = originalStdout
			lang.PopThreadBindings()
			<-done
			return err
		}
	}

	// Get the namespace from the environment
	ns := env.CurrentNamespace()
	if ns == nil {
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		log.Fatalf("Failed to get current namespace")
	}

	dashMain := ns.FindInternedVar(lang.NewSymbol("-main"))
	hasMainDashMain :=
		ns.Name().Name() == "main" &&
		dashMain != nil &&
		!dashMain.IsMacro()

	// Generate Go code to a buffer
	var codegenBuffer bytes.Buffer
	gen := NewGenerator(&codegenBuffer)

	if hasMainDashMain {
		gen.addImport("os")
		gen.addImportUnderscore("github.com/glojurelang/glojure/pkg/glj")
		gen.addImportUnderscore("github.com/glojurelang/glojure/pkg/stdlib/clojure/core")
	}

	if err := gen.Generate(ns); err != nil {
		stdoutWrite.Close()
		os.Stdout = originalStdout
		lang.PopThreadBindings()
		<-done
		return err
	}

	stdoutWrite.Close()
	os.Stdout = originalStdout
	lang.PopThreadBindings()
	<-done

	// Transform into main package format only if main/-main exists
	var outputCode []byte = codegenBuffer.Bytes()
	if hasMainDashMain {
		code := string(outputCode) + `

func main() {
	LoadNS()

	args := os.Args[1:]
	argsAny := make([]any, len(args))
	for i, s := range args { argsAny[i] = s }

	ns := lang.FindNamespace(lang.NewSymbol("main"))
	dashMain := ns.FindInternedVar(lang.NewSymbol("-main"))
	if dashMain != nil && !dashMain.IsMacro() {
		mainFn := dashMain.Get()
		lang.Apply(mainFn, argsAny)
	}
}
`

		outputCode = []byte(code)
	}

	// Emit the AOT code
	os.Stdout.Write(outputCode)
	return nil
}

// Compile a Clojure file to Go code using AOT compilation
func CompileAOTFile(inputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return CompileAOTString(string(content), inputFile)
}
