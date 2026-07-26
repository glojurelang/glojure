// Package interactive supplies the optional terminal, nREPL, and socket REPL
// commands used by gljmain. Import it for side effects in a full glj command.
package interactive

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/glojurelang/glojure/pkg/gljmain"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/nrepl"
	"github.com/glojurelang/glojure/pkg/repl"
	"github.com/glojurelang/glojure/pkg/srepl"
)

type commands struct{}

func init() {
	gljmain.RegisterInteractiveCommands(commands{})
}

func (commands) StartREPL() {
	repl.Start()
}

func (commands) StartNREPL(arg string) {
	host, port, portFile := parseServerArg(arg, "--nrepl")
	srv, err := nrepl.Start(host, port, portFile)
	if err != nil {
		log.Fatal(err)
	}

	actualPort := srv.Port()
	fmt.Printf("nREPL server started on port %d on host %s - nrepl://%s:%d\n",
		actualPort, host, host, actualPort)

	go srv.Serve()
	waitForShutdown("\nnREPL server shutting down...", srv.Stop)
}

func (commands) StartSREPL(arg string) {
	host, port, portFile := parseServerArg(arg, "--srepl")
	srv, err := srepl.Start(host, port, portFile)
	if err != nil {
		log.Fatal(err)
	}

	actualPort := srv.Port()
	fmt.Printf("Socket REPL started on port %d on host %s - %s:%d\n",
		actualPort, host, host, actualPort)

	go srv.Serve()
	waitForShutdown("\nSocket REPL shutting down...", srv.Stop)
}

func (commands) ConnectNREPL(args []string) {
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
		opts := []repl.Option{repl.WithNREPLClient(client)}
		if histFile != "" {
			opts = append(opts, repl.WithHistoryFile(histFile, histFmt))
		}
		repl.Start(opts...)
		return
	}

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

func (commands) Color() {
	if err := color(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func color(in io.Reader, out io.Writer) error {
	input, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(
		out,
		repl.ColorSyntax([]rune(string(input)), lang.GlobalEnv),
	)
	return err
}

func waitForShutdown(message string, stop func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println(message)
	stop()
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
// --nrepl=VALUE or --srepl=VALUE. The prefix is e.g. "--nrepl".
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
