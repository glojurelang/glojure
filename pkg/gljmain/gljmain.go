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
	Mode         string   // "repl", "version", "help", "eval", "file"
	Expression   string   // for eval mode
	Filename     string   // for file mode
	IncludePaths []string // for -I flags
	CommandArgs  []string // remaining arguments after parsing
}

// parseArgs parses command line arguments and returns an Args struct
func parseArgs(args []string) (*Args, error) {
	if len(args) == 0 {
		return &Args{Mode: "repl"}, nil
	}

	// First pass: collect all -I flags and their paths
	var includePaths []string
	var remainingArgs []string
	var mode string
	var expression string
	var filename string

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--version":
			if mode == "" {
				mode = "version"
			}
			i++
		case "--help", "-h":
			if mode == "" {
				mode = "help"
			}
			i++
		case "-e":
			if mode == "" {
				mode = "eval"
				if i+1 >= len(args) {
					return nil, fmt.Errorf("glj: -e requires an expression")
				}
				expression = args[i+1]
				i += 2
			} else {
				remainingArgs = append(remainingArgs, args[i])
				i++
			}
		case "-I":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("glj: -I requires a path")
			}
			includePaths = append(includePaths, args[i+1])
			i += 2
		default:
			if mode == "" {
				mode = "file"
				filename = args[i]
			} else {
				remainingArgs = append(remainingArgs, args[i])
			}
			i++
		}
	}

	// If no explicit mode was set, default to repl
	if mode == "" {
		mode = "repl"
	}

	return &Args{
		Mode:         mode,
		Expression:   expression,
		Filename:     filename,
		IncludePaths: includePaths,
		CommandArgs:  remainingArgs,
	}, nil
}

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>        Evaluate expression from command line
  -I <path>        Add directory to front of library search path

  --version        Show version information
  -h, --help       Show this help message

Environment Variables:
  GLJPATH          PATH of directories for .glj libraries

Examples:
  glj                   # Start REPL
  glj -I /path/to/lib   # Add library path and start REPL
  glj -I /lib1 -I /lib2 # Add multiple library paths
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

	// Process -I include paths first (add to front of load path)
	// Process in reverse order so first -I on command line gets highest priority
	for i := len(parsedArgs.IncludePaths) - 1; i >= 0; i-- {
		path := parsedArgs.IncludePaths[i]
		if path != "" {
			// Skip non-existent path directories
			if _, err := os.Stat(path); err == nil {
				runtime.AddLoadPath(os.DirFS(path), true)
			}
		}
	}

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
