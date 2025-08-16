package gljmain

import (
	"bufio"
	"flag"
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
	fs := flag.NewFlagSet("glj", flag.ExitOnError)

	var (
		helpFlag    = fs.Bool("help", false, "Show this help message")
		versionFlag = fs.Bool("version", false, "Show version information")
		evalFlag    = fs.String("e", "", "Evaluate expression from command line")
	)

	fs.Bool("h", false, "Show this help message (shorthand)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *helpFlag || fs.Lookup("h").Value.String() == "true" {
		return &Args{Mode: "help"}, nil
	}

	if *versionFlag {
		return &Args{Mode: "version"}, nil
	}

	if *evalFlag != "" {
		return &Args{
			Mode:        "eval",
			Expression:  *evalFlag,
			CommandArgs: fs.Args(),
		}, nil
	}

	remainingArgs := fs.Args()
	if len(remainingArgs) > 0 {
		return &Args{
			Mode:        "file",
			Filename:    remainingArgs[0],
			CommandArgs: remainingArgs[1:],
		}, nil
	}

	return &Args{Mode: "repl"}, nil
}

// setupLoadPaths configures the library search path
func setupLoadPaths() {
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

	// Add current directory to end of load path
	runtime.AddLoadPath(os.DirFS("."), false)
}

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>        Evaluate expression from command line

  --version        Show version information
  -h, --help       Show this help message

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
	parsedArgs, err := parseArgs(args)
	if err != nil {
		log.Fatal(err)
	}

	// Setup library search paths
	setupLoadPaths()

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
