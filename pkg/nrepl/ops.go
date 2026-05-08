package nrepl

import (
	"fmt"
	"net"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/reader"
	"github.com/gloathub/glojure/pkg/runtime"
)

func newSessionID() string {
	return uuid.New().String()
}

func (s *Server) opClone(msg map[string]interface{}, conn net.Conn) {
	sess := s.createSession()
	// If cloning from an existing session, inherit its namespace.
	if srcID := msgStr(msg, "session"); srcID != "" {
		if src := s.getSession(srcID); src != nil {
			sess.NS = src.NS
		}
	}
	sendMsg(conn, map[string]interface{}{
		"id":          msg["id"],
		"status":      []interface{}{"done"},
		"new-session": sess.ID,
	})
}

func (s *Server) opClose(msg map[string]interface{}, conn net.Conn) {
	sessionID := msgStr(msg, "session")
	if sessionID != "" {
		s.removeSession(sessionID)
	}
	sendMsg(conn, map[string]interface{}{
		"id":      msg["id"],
		"session": sessionID,
		"status":  []interface{}{"done"},
	})
}

func (s *Server) opDescribe(msg map[string]interface{}, conn net.Conn) {
	sendMsg(conn, map[string]interface{}{
		"id":      msg["id"],
		"session": msgStr(msg, "session"),
		"ops": map[string]interface{}{
			"clone":       map[string]interface{}{},
			"close":       map[string]interface{}{},
			"describe":    map[string]interface{}{},
			"eval":        map[string]interface{}{},
			"completions": map[string]interface{}{},
			"interrupt":   map[string]interface{}{},
			"load-file":   map[string]interface{}{},
			"ls-sessions": map[string]interface{}{},
		},
		"versions": map[string]interface{}{
			"glojure": map[string]interface{}{
				"version-string": runtime.Version,
			},
			"nrepl": map[string]interface{}{
				"version-string": "0.1.0",
				"major":          int64(0),
				"minor":          int64(1),
				"incremental":    int64(0),
			},
		},
		"status": []interface{}{"done"},
	})
}

func (s *Server) opEval(msg map[string]interface{}, conn net.Conn) {
	code := msgStr(msg, "code")
	sessionID := msgStr(msg, "session")
	msgID := msg["id"]

	sess := s.getOrCreateSession(sessionID)
	sessionID = sess.ID

	// Resolve the namespace for this session.
	if nsStr := msgStr(msg, "ns"); nsStr != "" {
		sess.NS = nsStr
	}
	ns := lang.FindNamespace(lang.NewSymbol(sess.NS))
	if ns == nil {
		ns = lang.FindNamespace(lang.NewSymbol("user"))
	}

	if ns == nil {
		// Create the user namespace if it doesn't exist yet.
		ns = lang.FindOrCreateNamespace(lang.NewSymbol(sess.NS))
	}

	// Create a writer that sends "out" messages over the nREPL connection.
	outWriter := &nreplWriter{
		conn:      conn,
		id:        msgID,
		sessionID: sessionID,
		key:       "out",
	}
	// Push thread bindings for *ns*, *out*, and other dynamic vars
	// that the evaluator expects (same set as the REPL's initEnv).
	bindings := lang.NewMap(
		lang.VarCurrentNS, ns,
		lang.VarOut, outWriter,
		lang.VarWarnOnReflection, lang.VarWarnOnReflection.Deref(),
		lang.VarUncheckedMath, lang.VarUncheckedMath.Deref(),
		lang.VarDataReaders, lang.VarDataReaders.Deref(),
	)

	var lastValue string
	var evalErr error

	func() {
		lang.PushThreadBindings(bindings)
		defer lang.PopThreadBindings()
		defer func() {
			if r := recover(); r != nil {
				evalErr = fmt.Errorf("panic: %v\nstacktrace:\n%s", r, string(debug.Stack()))
			}
		}()

		env := lang.GlobalEnv
		rdr := reader.New(
			strings.NewReader(code),
			reader.WithFilename("nrepl"),
			reader.WithGetCurrentNS(func() *lang.Namespace {
				return env.CurrentNamespace()
			}),
		)
		vals, err := rdr.ReadAll()
		if err != nil {
			evalErr = err
			return
		}
		for _, val := range vals {
			result, err := env.Eval(val)
			if err != nil {
				evalErr = err
				return
			}
			lastValue = lang.PrintString(result)
		}
		// Update session namespace from current thread binding.
		sess.NS = env.CurrentNamespace().Name().String()
	}()

	// Flush any buffered output.
	outWriter.flush()

	if evalErr != nil {
		sendMsg(conn, map[string]interface{}{
			"id":      msgID,
			"session": sessionID,
			"err":     evalErr.Error() + "\n",
		})
		sendMsg(conn, map[string]interface{}{
			"id":      msgID,
			"session": sessionID,
			"status":  []interface{}{"eval-error", "done"},
			"ex":      evalErr.Error(),
		})
		return
	}

	sendMsg(conn, map[string]interface{}{
		"id":      msgID,
		"session": sessionID,
		"value":   lastValue,
		"ns":      sess.NS,
	})
	sendMsg(conn, map[string]interface{}{
		"id":      msgID,
		"session": sessionID,
		"status":  []interface{}{"done"},
	})
}

func (s *Server) opCompletions(msg map[string]interface{}, conn net.Conn) {
	prefix := msgStr(msg, "prefix")
	sessionID := msgStr(msg, "session")

	sess := s.getOrCreateSession(sessionID)

	ns := lang.FindNamespace(lang.NewSymbol(sess.NS))
	if ns == nil {
		ns = lang.FindNamespace(lang.NewSymbol("user"))
	}

	var completions []interface{}

	// Complete from current namespace mappings.
	if mappings := ns.Mappings(); mappings != nil {
		for seq := lang.Seq(mappings); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			sym := entry.Key().(*lang.Symbol)
			name := sym.Name()
			if strings.HasPrefix(name, prefix) {
				comp := map[string]interface{}{
					"candidate": name,
				}
				if vr, ok := entry.Val().(*lang.Var); ok {
					comp["ns"] = vr.Namespace().Name().String()
				}
				completions = append(completions, comp)
			}
		}
	}

	// Complete namespace names.
	for nsSeq := lang.AllNamespaces(); nsSeq != nil; nsSeq = nsSeq.Next() {
		nsObj := nsSeq.First().(*lang.Namespace)
		name := nsObj.Name().String()
		if strings.HasPrefix(name, prefix) {
			completions = append(completions, map[string]interface{}{
				"candidate": name,
				"type":      "namespace",
			})
		}
	}

	if completions == nil {
		completions = []interface{}{}
	}

	sendMsg(conn, map[string]interface{}{
		"id":          msg["id"],
		"session":     sess.ID,
		"completions": completions,
		"status":      []interface{}{"done"},
	})
}

func (s *Server) opInterrupt(msg map[string]interface{}, conn net.Conn) {
	// Stub: no real interrupt support yet.
	sendMsg(conn, map[string]interface{}{
		"id":      msg["id"],
		"session": msgStr(msg, "session"),
		"status":  []interface{}{"done"},
	})
}

func (s *Server) opLoadFile(msg map[string]interface{}, conn net.Conn) {
	// Treat load-file as eval of the file content.
	file := msgStr(msg, "file")
	if file != "" {
		msg["code"] = file
	}
	s.opEval(msg, conn)
}

func (s *Server) opLsSessions(msg map[string]interface{}, conn net.Conn) {
	s.mu.RLock()
	ids := make([]interface{}, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	sendMsg(conn, map[string]interface{}{
		"id":       msg["id"],
		"sessions": ids,
		"status":   []interface{}{"done"},
	})
}

// nreplWriter sends writes as nREPL "out" or "err" messages.
type nreplWriter struct {
	conn      net.Conn
	id        interface{}
	sessionID string
	key       string // "out" or "err"
	buf       strings.Builder
}

func (w *nreplWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	// Flush on newlines for responsive output.
	if strings.ContainsRune(string(p), '\n') {
		w.flush()
	}
	return len(p), nil
}

func (w *nreplWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	sendMsg(w.conn, map[string]interface{}{
		"id":      w.id,
		"session": w.sessionID,
		w.key:     w.buf.String(),
	})
	w.buf.Reset()
}
