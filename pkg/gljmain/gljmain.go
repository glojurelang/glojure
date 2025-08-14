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

// Args represents the parsed command line arguments
type Args struct {
	Mode        string   // "repl", "version", "help", "eval", "file"
	Expression  string   // for eval mode
	Filename    string   // for file mode
	CommandArgs []string // remaining arguments after parsing
}

// parseArgs parses command line arguments and returns an Args struct
func parseArgs(args []string) (*Args, error) {
	if len(args) == 0 {
		return &Args{Mode: "repl"}, nil
	}

	switch args[0] {
	case "--version":
		return &Args{Mode: "version"}, nil
	case "--help", "-h":
		return &Args{Mode: "help"}, nil
	case "-e":
		if len(args) < 2 {
			return nil, fmt.Errorf("glj: -e requires an expression")
		}
		return &Args{
			Mode:        "eval",
			Expression:  args[1],
			CommandArgs: args[2:],
		}, nil
	default:
		return &Args{
			Mode:        "file",
			Filename:    args[0],
			CommandArgs: args[1:],
		}, nil
	}
}

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>        Evaluate expression from command line
  -h, --help       Show this help message
  --version        Show version information

Environment Variables:
  GLJPATH          PATH of directories for .glj libraries

Examples:
  glj                   # Start REPL
  glj -e "(+ 1 2)"      # Evaluate expression
  glj script.glj        # Run script file
  glj --version         # Show version
  glj --help            # Show this help

For more information, visit: https://github.com/glojurelang/glojure
`, runtime.VERSION)
}

func Main(args []string) {
	// Add current directory to end of load path
	runtime.AddLoadPath(os.DirFS("."), false)

	// Add GLJPATH directories to front of load path if set
	loadPaths := os.Getenv("GLJPATH")
	if loadPaths != "" {
		paths := strings.Split(loadPaths, ":")
		for i := len(paths) - 1; i >= 0; i-- {
			path := paths[i]
			if path != "" {
				// Skip non-existent path directories
				if _, err := os.Stat(path); err == nil {
					runtime.AddLoadPath(os.DirFS(path), true)
				}
			}
		}
	}

	parsedArgs, err := parseArgs(args)
	if err != nil {
		log.Fatal(err)
	}

	switch parsedArgs.Mode {
	case "repl":
		repl.Start()
	case "version":
		fmt.Printf("glojure v%s\n", runtime.VERSION)
		return
	case "help":
		printHelp()
		return
	case "eval":
		// Evaluate expression from command line
		env := lang.GlobalEnv

		// Set command line args (everything after -e and the expression)
		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).
			BindRoot(lang.Seq(parsedArgs.CommandArgs))

		rdr := reader.New(
			strings.NewReader(parsedArgs.Expression),
			reader.WithGetCurrentNS(func() *lang.Namespace {
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
	case "file":
		// Execute file
		file, err := os.Open(parsedArgs.Filename)
		if err != nil {
			log.Fatal(err)
		}
		env := lang.GlobalEnv

		core := lang.FindNamespace(lang.NewSymbol("glojure.core"))
		core.FindInternedVar(lang.NewSymbol("*command-line-args*")).
			BindRoot(lang.Seq(parsedArgs.CommandArgs))

		rdr := reader.New(
			bufio.NewReader(file),
			reader.WithGetCurrentNS(func() *lang.Namespace {
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
