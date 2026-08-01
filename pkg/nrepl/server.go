package nrepl

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/glojurelang/glojure/pkg/lang"
)

// Server is a minimal nREPL server for editor integration.
type Server struct {
	listener net.Listener
	sessions map[string]*Session
	mu       sync.RWMutex
	done     chan struct{}
	portFile string
	wg       sync.WaitGroup
}

// Session tracks per-session state for an nREPL client.
type Session struct {
	ID string
	NS string // current namespace name
}

// Start creates and starts an nREPL server on the given host and port.
// Port 0 means auto-assign a free port.
func Start(host string, port int, portFile string) (*Server, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("nrepl: listen %s: %w", addr, err)
	}

	// Ensure the user namespace is properly initialized.
	initUserNS()

	s := &Server{
		listener: ln,
		sessions: make(map[string]*Session),
		done:     make(chan struct{}),
		portFile: portFile,
	}

	if portFile != "" {
		dir := filepath.Dir(portFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			ln.Close()
			return nil, fmt.Errorf("nrepl: create port file dir: %w", err)
		}
		actualPort := s.Port()
		if err := os.WriteFile(portFile, []byte(strconv.Itoa(actualPort)), 0644); err != nil {
			ln.Close()
			return nil, fmt.Errorf("nrepl: write port file: %w", err)
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
				fmt.Fprintf(os.Stderr, "nrepl: accept error: %v\n", err)
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

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	br := newByteReader(conn)
	for {
		val, err := bencodeRead(br)
		if err != nil {
			return // connection closed or read error
		}
		msg, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		s.dispatch(msg, conn)
	}
}

func (s *Server) dispatch(msg map[string]interface{}, conn net.Conn) {
	op, _ := msg["op"].(string)
	switch op {
	case "clone":
		s.opClone(msg, conn)
	case "close":
		s.opClose(msg, conn)
	case "describe":
		s.opDescribe(msg, conn)
	case "eval":
		s.opEval(msg, conn)
	case "completions":
		s.opCompletions(msg, conn)
	case "info":
		s.opInfo(msg, conn)
	case "interrupt":
		s.opInterrupt(msg, conn)
	case "load-file":
		s.opLoadFile(msg, conn)
	case "ls-sessions":
		s.opLsSessions(msg, conn)
	default:
		sendMsg(conn, map[string]interface{}{
			"id":     msg["id"],
			"status": []interface{}{"error", "unknown-op", "done"},
		})
	}
}

func (s *Server) getSession(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

func (s *Server) getOrCreateSession(sessionID string) *Session {
	if sessionID != "" {
		if sess := s.getSession(sessionID); sess != nil {
			return sess
		}
	}
	return s.createSession()
}

func (s *Server) createSession() *Session {
	sess := &Session{
		ID: newSessionID(),
		NS: "user",
	}
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()
	return sess
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

func (s *Server) removeSession(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func sendMsg(conn net.Conn, msg map[string]interface{}) {
	// Strip nil values -- bencode can't encode nil.
	for k, v := range msg {
		if v == nil {
			delete(msg, k)
		}
	}
	data, err := BencodeEncode(msg)
	if err != nil {
		return
	}
	conn.Write(data)
}

func msgStr(msg map[string]interface{}, key string) string {
	v, _ := msg[key].(string)
	return v
}
