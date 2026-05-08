package gljmain

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	// bootstrap the runtime
	_ "github.com/gloathub/glojure/pkg/glj"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/nrepl"
	"github.com/gloathub/glojure/pkg/reader"
	"github.com/gloathub/glojure/pkg/repl"
	"github.com/gloathub/glojure/pkg/runtime"
)

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>        Evaluate expression from command line
  --nrepl          Start an nREPL server
  --port <port>    Port for nREPL server (default: auto)
  -h, --help       Show this help message
  --version        Show version information

Examples:
  glj                        # Start REPL
  glj -e "(+ 1 2)"           # Evaluate expression
  glj script.glj             # Run script file
  glj --nrepl                # Start nREPL on random port
  glj --nrepl --port 7888    # Start nREPL on port 7888
  glj --version              # Show version
  glj --help                 # Show this help

For more information, visit: https://github.com/gloathub/glojure
`, runtime.Version)
}

func Main(args []string) {
	runtime.AddLoadPath(os.DirFS("."))

	if len(args) == 0 {
		// Check if stdin is a terminal
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			// Interactive terminal: start REPL
			repl.Start()
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
	} else if args[0] == "--nrepl" {
		startNREPL(args[1:])
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

func startNREPL(args []string) {
	host := "localhost"
	port := 0
	portFile := ".gloat/.nrepl-port"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				log.Fatal("glj: --port requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				log.Fatalf("glj: invalid port: %s", args[i])
			}
			port = p
		case "--host":
			if i+1 >= len(args) {
				log.Fatal("glj: --host requires a value")
			}
			i++
			host = args[i]
		case "--port-file":
			if i+1 >= len(args) {
				log.Fatal("glj: --port-file requires a value")
			}
			i++
			portFile = args[i]
		default:
			log.Fatalf("glj: unknown nrepl option: %s", args[i])
		}
	}

	srv, err := nrepl.Start(host, port, portFile)
	if err != nil {
		log.Fatal(err)
	}

	actualPort := srv.Port()
	fmt.Printf("nREPL server started on port %d on host %s - nrepl://%s:%d\n",
		actualPort, host, host, actualPort)

	// Serve in background, wait for signal to shut down.
	go srv.Serve()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nnREPL server shutting down...")
	srv.Stop()
}
