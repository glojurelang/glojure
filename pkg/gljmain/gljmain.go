package gljmain

import (
	"fmt"
	"io"
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
  -h, --help       Show this help message
  --version        Show version information

Examples:
  glj                   # Start REPL
  glj -e "(+ 1 2)"      # Evaluate expression
  glj script.glj        # Run script file
  glj --version         # Show version
  glj --help            # Show this help

Scripts with shebang lines (#!/usr/bin/env glj) are supported.

For more information, visit: https://github.com/glojurelang/glojure
`, runtime.VERSION)
}

func Main(args []string) {
	runtime.AddLoadPath(os.DirFS("."))

	if len(args) == 0 {
		repl.Start()
	} else if args[0] == "--version" {
		fmt.Printf("glojure v%s\n", runtime.VERSION)
		return
	} else if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	} else if args[0] == "-e" {
		// Evaluate expression from command line
		if len(args) < 2 {
			log.Fatal("glj: -e requires an expression")
		}
		expr := args[1]
		env := lang.GlobalEnv

		// Set command line args (everything after -e and the expression)
		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
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
		defer file.Close()

		env := lang.GlobalEnv

		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).BindRoot(lang.Seq(args[1:]))

		// Read the entire file content and filter out shebang lines
		content, err := io.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}

		// Remove shebang line if present
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#!") {
			lines = lines[1:]
		}

		// Create a new reader from the filtered content
		filteredContent := strings.Join(lines, "\n")
		if filteredContent != "" && !strings.HasSuffix(filteredContent, "\n") {
			filteredContent += "\n"
		}

		rdr := reader.New(strings.NewReader(filteredContent), reader.WithGetCurrentNS(func() *lang.Namespace {
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
