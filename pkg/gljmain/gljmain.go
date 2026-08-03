package gljmain

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// bootstrap the runtime
	_ "github.com/glojurelang/glojure/pkg/glj"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/runtime"
)

// InteractiveCommands supplies the optional terminal and network REPL
// commands. Programs that need them should import pkg/gljmain/interactive.
type InteractiveCommands interface {
	StartREPL()
	StartNREPL(arg string)
	StartSREPL(arg string)
	ConnectNREPL(args []string)
	Color()
}

var registeredInteractiveCommands InteractiveCommands

// RegisterInteractiveCommands installs the optional interactive command
// implementation. It is intended to be called once, from package init.
func RegisterInteractiveCommands(commands InteractiveCommands) {
	if commands == nil {
		panic("gljmain: cannot register nil interactive commands")
	}
	if registeredInteractiveCommands != nil {
		panic("gljmain: interactive commands already registered")
	}
	registeredInteractiveCommands = commands
}

func interactiveCommands() InteractiveCommands {
	if registeredInteractiveCommands == nil {
		log.Fatal("glj: interactive commands are unavailable; import github.com/glojurelang/glojure/pkg/gljmain/interactive")
	}
	return registeredInteractiveCommands
}

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -Sdeps <edn>          Merge inline deps data after the project deps.edn
  -e <expr>              Evaluate expression from command line
  --nrepl[=VALUE]        Start nREPL server
  --nrepl-connect H:P    Connect REPL to nREPL server
  --srepl[=VALUE]        Start socket REPL server
  --color                Syntax highlight stdin with ANSI colors
  -h, --help             Show this help message
  --version              Show version information

A deps.edn in the current directory is resolved before evaluating code,
running a file, or starting a REPL or REPL server.

Examples:
  glj                           # Start REPL
  glj -e "(+ 1 2)"              # Evaluate expression
  glj script.glj                # Run script file
  glj --nrepl                   # Start nREPL on random port
  glj --nrepl=7888              # Start nREPL on port 7888
  glj --nrepl=0.0.0.0:7888      # Bind to all interfaces
  glj --nrepl=.nrepl-port       # Write port to file
  glj --srepl                   # Start socket REPL on random port
  glj --srepl=7777              # Start socket REPL on port 7777
  glj --color < file.clj         # Syntax highlight Clojure code
  glj --version                 # Show version
  glj --help                    # Show this help

For more information, visit: https://github.com/glojurelang/glojure
`, runtime.Version)
}

func usesProjectDeps(args []string) bool {
	if len(args) == 0 {
		return true
	}
	arg := args[0]
	return arg == "-e" || arg == "--nrepl" || strings.HasPrefix(arg, "--nrepl=") ||
		arg == "--srepl" || strings.HasPrefix(arg, "--srepl=") ||
		!strings.HasPrefix(arg, "-")
}

func splitDepsOption(args []string) (string, []string, error) {
	if len(args) == 0 || args[0] != "-Sdeps" {
		return "", args, nil
	}
	if len(args) < 2 {
		return "", nil, fmt.Errorf("glj: -Sdeps requires an EDN map")
	}
	return args[1], args[2:], nil
}

func loadProjectDeps(extraEDN string) {
	const path = "deps.edn"
	projectEDN := "{}"
	if content, err := os.ReadFile(path); err == nil {
		projectEDN = string(content)
	} else {
		if os.IsNotExist(err) {
			if extraEDN == "" {
				return
			}
		} else {
			log.Fatal(err)
		}
	}
	if extraEDN == "" {
		extraEDN = "{}"
	}
	runtime.ReadEval(
		fmt.Sprintf(
			`(require 'clojurestar.deps)
             (let [merge-value (fn [left right]
                                 (if (and (map? left) (map? right))
                                   (merge left right)
                                   right))]
               (clojurestar.deps/add-deps
                (merge-with merge-value
                            (read-string %s)
                            (read-string %s))))`,
			strconv.Quote(projectEDN), strconv.Quote(extraEDN)),
		runtime.WithFilename(path),
	)
}

func Main(args []string) {
	for _, path := range filepath.SplitList(os.Getenv("GLJ_CLASSPATH")) {
		if path != "" {
			runtime.AddLoadPath(os.DirFS(path))
		}
	}
	extraEDN, args, err := splitDepsOption(args)
	if err != nil {
		log.Fatal(err)
	}
	if usesProjectDeps(args) {
		loadProjectDeps(extraEDN)
	}

	if len(args) == 0 {
		// Check if stdin is a terminal
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			// Interactive terminal: start REPL
			interactiveCommands().StartREPL()
		} else {
			// Piped input: evaluate and exit with proper error handling
			env := lang.GlobalEnv
			rdr := reader.New(bufio.NewReader(os.Stdin), reader.WithGetCurrentNS(func() *lang.Namespace {
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
				result, err := env.Eval(val)
				if err != nil {
					log.Fatal(err)
				}
				if !lang.IsNil(result) {
					fmt.Println(lang.PrintString(result))
				}
			}
		}
	} else if args[0] == "--version" {
		fmt.Printf("glojure v%s\n", runtime.Version)
		return
	} else if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	} else if args[0] == "--nrepl" || strings.HasPrefix(args[0], "--nrepl=") {
		interactiveCommands().StartNREPL(args[0])
		return
	} else if args[0] == "--srepl" || strings.HasPrefix(args[0], "--srepl=") {
		interactiveCommands().StartSREPL(args[0])
		return
	} else if args[0] == "--nrepl-connect" {
		interactiveCommands().ConnectNREPL(args)
		return
	} else if args[0] == "--color" {
		interactiveCommands().Color()
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
	} else if strings.HasPrefix(args[0], "-") {
		log.Fatalf("glj: unknown option: %s\nRun 'glj --help' for usage.", args[0])
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
