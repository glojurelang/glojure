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
  --aot            Compile Glojure file to Go code (stdout)
  -h, --help       Show this help message
  --version        Show version information

Environment Variables:
  GLJPATH          Colon-separated PATH to search for .glj libraries

Examples:
  glj                              # Start a Glojure REPL
  glj script.glj          	       # Run script file
  glj -e '(* 6 7)'                 # Evaluate expression
  glj --aot file.glj               # Compile .glj file to Go code
  glj --aot -e '(ns main)(defn …)' # Compile expression to Go code
  glj --version                    # Show version
  glj --help                       # Show this help

For more information, visit: https://github.com/glojurelang/glojure
`, runtime.Version)
}

func Main(args []string) {
	if len(args) == 0 {
		repl.Start()
	} else if args[0] == "--version" {
		fmt.Printf("glojure v%s\n", runtime.Version)
		return
	} else if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	} else if args[0] == "--aot" {
		// AOT compile file or expression
		if len(args) < 2 {
			log.Fatal("glj: --aot requires a file path or -e expression")
		}

		if args[1] == "-e" {
			// AOT compile expression
			if len(args) < 3 {
				log.Fatal("glj: --aot -e requires an expression")
			}
			expr := args[2]
			if err := runtime.CompileAOTString(expr, "<string>"); err != nil {
				log.Fatal(err)
			}
		} else {
			// AOT compile file
			inputFile := args[1]
			if err := runtime.CompileAOTFile(inputFile); err != nil {
				log.Fatal(err)
			}
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
