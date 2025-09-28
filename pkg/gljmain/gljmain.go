package gljmain

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	// bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/repl"
	"github.com/glojurelang/glojure/pkg/runtime"
)

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>        Evaluate expression from command line
  -m <namespace>   Run -main function of namespace with args
  -h, --help       Show this help message
  --version        Show version information

Examples:
  glj                             # Start REPL
  glj -e "(+ 1 2)"                # Evaluate expression
  glj -m glojure.test-runner      # Run test runner
  glj script.glj                  # Run script file
  glj --version                   # Show version
  glj --help                      # Show this help

For more information, visit: https://github.com/glojurelang/glojure
`, runtime.Version)
}

func Main(args []string) {
	runtime.AddLoadPath(os.DirFS("."))

	if len(args) == 0 {
		repl.Start()
	} else if args[0] == "--version" {
		fmt.Printf("glojure v%s\n", runtime.Version)
		return
	} else if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	} else if args[0] == "-m" || args[0] == "--main" {
		// Call the -main function from a namespace with string arguments
		if len(args) < 2 {
			log.Fatal("glj: -m requires a namespace name")
		}
		mainNS := args[1]
		mainArgs := args[2:]
		
		// Set command line args
		core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).BindRoot(lang.Seq(mainArgs))
		
		// Require the namespace  
		nsSym := lang.NewSymbol(mainNS)
		requireVar := core.FindInternedVar(lang.NewSymbol("require"))
		result := requireVar.Invoke(nsSym)
		if err, ok := result.(error); ok {
			log.Fatalf("Failed to require namespace %s: %v", mainNS, err)
		}
		
		// Find the namespace
		ns := lang.FindNamespace(nsSym)
		if ns == nil {
			log.Fatalf("Namespace %s not found after require", mainNS)
		}
		
		// Find and call the -main function
		mainVar := ns.FindInternedVar(lang.NewSymbol("-main"))
		if mainVar == nil {
			log.Fatalf("No -main function found in namespace %s", mainNS)
		}
		
		// Convert args to Clojure values
		var clojureArgs []interface{}
		for _, arg := range mainArgs {
			clojureArgs = append(clojureArgs, arg)
		}
		
		// Call -main with the arguments
		result = mainVar.Invoke(clojureArgs...)
		if err, ok := result.(error); ok {
			log.Fatalf("Error calling -main: %v", err)
		}
		return
	} else if args[0] == "-e" {
		// Evaluate expression from command line
		if len(args) < 2 {
			log.Fatal("glj: -e requires an expression")
		}
		expr := args[1]
		env := lang.GlobalEnv

		// Set command line args (everything after -e and the expression)
		core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).BindRoot(lang.Seq(args[2:]))

		rdr := reader.New(strings.NewReader(expr), reader.WithGetCurrentNS(func() *lang.Namespace {
			return env.CurrentNamespace()
		}))
		var lastResult interface{}
		for {
			val, err := rdr.ReadOne()
			if err == reader.ErrEOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			result, err := env.Eval(val)
			if err != nil {
				log.Fatal(err)
			}
			lastResult = result
		}
		// Print only the final result unless it's nil
		if !lang.IsNil(lastResult) {
			fmt.Println(lang.PrintString(lastResult))
		}
	} else {
		// Execute file
		file, err := os.Open(args[0])
		if err != nil {
			log.Fatal(err)
		}
		env := lang.GlobalEnv

		core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).BindRoot(lang.Seq(args[1:]))

		rdr := reader.New(bufio.NewReader(file), reader.WithGetCurrentNS(func() *lang.Namespace {
			return env.CurrentNamespace()
		}))
		for {
			val, err := rdr.ReadOne()
			if err == reader.ErrEOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			_, err = env.Eval(val)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
