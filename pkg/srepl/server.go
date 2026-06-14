package srepl

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/glojurelang/glojure/pkg/lang"
)

// Server is a plain-text socket REPL server.
// Each connection gets an independent read-eval-print loop.
type Server struct {
	listener net.Listener
	done     chan struct{}
	portFile string
	wg       sync.WaitGroup
}

// Start creates and starts a socket REPL server on the given host and port.
// Port 0 means auto-assign a free port.
func Start(host string, port int, portFile string) (*Server, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("srepl: listen %s: %w", addr, err)
	}

	initUserNS()

	s := &Server{
		listener: ln,
		done:     make(chan struct{}),
		portFile: portFile,
	}

	if portFile != "" {
		dir := filepath.Dir(portFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			ln.Close()
			return nil, fmt.Errorf("srepl: create port file dir: %w", err)
		}
		actualPort := s.Port()
		if err := os.WriteFile(portFile, []byte(strconv.Itoa(actualPort)), 0644); err != nil {
			ln.Close()
			return nil, fmt.Errorf("srepl: write port file: %w", err)
		}
	}

	return s, nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Host returns the host the server is listening on.
func (s *Server) Host() string {
	return s.listener.Addr().(*net.TCPAddr).IP.String()
}

// Serve accepts connections in a loop. Blocks until Stop is called.
func (s *Server) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				fmt.Fprintf(os.Stderr, "srepl: accept error: %v\n", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// Stop shuts down the server and cleans up the port file.
func (s *Server) Stop() {
	close(s.done)
	s.listener.Close()
	s.wg.Wait()
	if s.portFile != "" {
		os.Remove(s.portFile)
	}
}

// initUserNS ensures the "user" namespace exists with clojure.core
// symbols referred, matching what (ns user) does in a standard REPL.
func initUserNS() {
	if lang.FindNamespace(lang.NewSymbol("user")) != nil {
		return
	}
	coreNS := lang.FindNamespace(lang.NewSymbol("clojure.core"))
	bindings := lang.NewMap(
		lang.VarCurrentNS, coreNS,
		lang.VarWarnOnReflection, lang.VarWarnOnReflection.Deref(),
		lang.VarUncheckedMath, lang.VarUncheckedMath.Deref(),
		lang.VarDataReaders, lang.VarDataReaders.Deref(),
	)
	lang.PushThreadBindings(bindings)
	defer lang.PopThreadBindings()
	lang.GlobalEnv.Eval(lang.NewList(
		lang.NewSymbol("ns"),
		lang.NewSymbol("user"),
	))
}
