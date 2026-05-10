package gljmain

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
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
	"github.com/gloathub/glojure/pkg/srepl"
)

func printHelp() {
	fmt.Printf(`Glojure v%s

Usage: glj [options] [file]

Options:
  -e <expr>              Evaluate expression from command line
  --nrepl[=VALUE]        Start nREPL server
  --nrepl-connect H:P    Connect REPL to nREPL server
  --srepl[=VALUE]        Start socket REPL server
  -h, --help             Show this help message
  --version              Show version information

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
  glj --version                 # Show version
  glj --help                    # Show this help

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
	} else if args[0] == "--nrepl" || strings.HasPrefix(args[0], "--nrepl=") {
		host, port, portFile := parseServerArg(args[0], "--nrepl")
		startNREPL(host, port, portFile)
		return
	} else if args[0] == "--srepl" || strings.HasPrefix(args[0], "--srepl=") {
		host, port, portFile := parseServerArg(args[0], "--srepl")
		startSREPL(host, port, portFile)
		return
	} else if args[0] == "--nrepl-connect" {
		if len(args) < 2 {
			log.Fatal("glj: --nrepl-connect requires HOST:PORT")
		}
		addr := args[1]
		idx := strings.LastIndex(addr, ":")
		if idx <= 0 || !isAllDigits(addr[idx+1:]) {
			log.Fatalf("glj: invalid address: %s (expected HOST:PORT)", addr)
		}
		host := addr[:idx]
		port, err := strconv.Atoi(addr[idx+1:])
		if err != nil {
			log.Fatalf("glj: invalid port: %s", addr[idx+1:])
		}
		client, err := nrepl.Connect(host, port)
		if err != nil {
			log.Fatalf("glj: failed to connect to nREPL at %s: %v", addr, err)
		}
		defer client.Close()

		// Parse optional --history FILE [--history-fmt FMT]
		var histFile, histFmt string
		for i := 2; i < len(args); i++ {
			if args[i] == "--history" && i+1 < len(args) {
				histFile = args[i+1]
				i++
			} else if args[i] == "--history-fmt" && i+1 < len(args) {
				histFmt = args[i+1]
				i++
			}
		}

		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			// Interactive terminal: full readline REPL
			opts := []repl.Option{repl.WithNREPLClient(client)}
			if histFile != "" {
				opts = append(opts, repl.WithHistoryFile(histFile, histFmt))
			}
			repl.Start(opts...)
		} else {
			// Piped input: eval and exit
			input, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			value, _, out, evalErr := client.Eval(string(input))
			if out != "" {
				fmt.Print(out)
			}
			if evalErr != nil {
				log.Fatal(evalErr)
			}
			if value != "" {
				fmt.Println(value)
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

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseServerArg extracts host, port, and port-file from a flag like
// --nrepl=VALUE or --srepl=VALUE.  The prefix is e.g. "--nrepl".
func parseServerArg(arg, prefix string) (host string, port int, portFile string) {
	host = "localhost"
	if !strings.HasPrefix(arg, prefix+"=") {
		return
	}
	val := strings.TrimPrefix(arg, prefix+"=")
	if isAllDigits(val) {
		p, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("glj: invalid port: %s", val)
		}
		port = p
	} else if idx := strings.LastIndex(val, ":"); idx > 0 && isAllDigits(val[idx+1:]) {
		host = val[:idx]
		p, err := strconv.Atoi(val[idx+1:])
		if err != nil {
			log.Fatalf("glj: invalid port: %s", val[idx+1:])
		}
		port = p
	} else if net.ParseIP(val) != nil {
		host = val
	} else {
		portFile = val
	}
	return
}

func startNREPL(host string, port int, portFile string) {
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

func startSREPL(host string, port int, portFile string) {
	srv, err := srepl.Start(host, port, portFile)
	if err != nil {
		log.Fatal(err)
	}

	actualPort := srv.Port()
	fmt.Printf("Socket REPL started on port %d on host %s - %s:%d\n",
		actualPort, host, host, actualPort)

	go srv.Serve()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nSocket REPL shutting down...")
	srv.Stop()
}
