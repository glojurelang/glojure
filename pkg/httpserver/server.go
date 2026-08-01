// Package httpserver adapts Clojure Ring handlers to Go's HTTP server.
package httpserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
)

// Server is a running Ring-compatible HTTP server.
type Server struct {
	server   *http.Server
	listener net.Listener
	done     chan error
	once     sync.Once
}

// Start listens on host:port and serves requests with handler. Port 0 asks the
// operating system to choose an available port.
func Start(handler interface{}, host string, port int) (*Server, error) {
	fn, ok := handler.(lang.IFn)
	if !ok {
		return nil, fmt.Errorf("httpserver: handler is not callable")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	s := &Server{listener: listener, done: make(chan error, 1)}
	s.server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveRing(fn, w, r, s.Port())
	})}
	go func() {
		err := s.server.Serve(listener)
		if err == http.ErrServerClosed {
			err = nil
		}
		s.done <- err
	}()
	return s, nil
}

// Host returns the bound listener host.
func (s *Server) Host() string {
	return s.listener.Addr().(*net.TCPAddr).IP.String()
}

// Port returns the bound listener port.
func (s *Server) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Stop gracefully stops the server.
func (s *Server) Stop() error {
	var err error
	s.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = s.server.Shutdown(ctx)
	})
	return err
}

// Wait blocks until the server stops.
func (s *Server) Wait() error {
	return <-s.done
}

func serveRing(handler lang.IFn, w http.ResponseWriter, r *http.Request, localPort int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			http.Error(w, fmt.Sprint(recovered), http.StatusInternalServerError)
		}
	}()
	host, portText, _ := net.SplitHostPort(r.Host)
	port, _ := strconv.Atoi(portText)
	if port == 0 {
		port = localPort
	}
	if host == "" {
		host = r.Host
	}
	remote, _, _ := net.SplitHostPort(r.RemoteAddr)
	headers := lang.NewMap()
	for name, values := range r.Header {
		headers = headers.Assoc(strings.ToLower(name), strings.Join(values, ",")).(lang.IPersistentMap)
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	request := lang.NewMap(
		lang.NewKeyword("server-port"), int64(port),
		lang.NewKeyword("server-name"), host,
		lang.NewKeyword("remote-addr"), remote,
		lang.NewKeyword("uri"), r.URL.Path,
		lang.NewKeyword("query-string"), nilIfEmpty(r.URL.RawQuery),
		lang.NewKeyword("scheme"), lang.NewKeyword(scheme),
		lang.NewKeyword("request-method"), lang.NewKeyword(strings.ToLower(r.Method)),
		lang.NewKeyword("protocol"), r.Proto,
		lang.NewKeyword("headers"), headers,
		lang.NewKeyword("body"), r.Body,
	)
	response := handler.Invoke(request)
	writeResponse(w, response)
}

func writeResponse(w http.ResponseWriter, response interface{}) {
	lookup, ok := response.(lang.ILookup)
	if !ok {
		http.Error(w, "Ring handler returned a non-map response", http.StatusInternalServerError)
		return
	}
	if headers, ok := lookup.ValAt(lang.NewKeyword("headers")).(lang.Seqable); ok {
		for entries := headers.Seq(); entries != nil; entries = entries.Next() {
			entry := entries.First().(lang.IMapEntry)
			w.Header().Set(lang.ToString(entry.Key()), lang.ToString(entry.Val()))
		}
	}
	status := 200
	if value := lookup.ValAt(lang.NewKeyword("status")); value != nil {
		status = int(lang.AsInt64(value))
	}
	w.WriteHeader(status)
	body := lookup.ValAt(lang.NewKeyword("body"))
	switch value := body.(type) {
	case nil:
	case string:
		_, _ = io.WriteString(w, value)
	case []byte:
		_, _ = w.Write(value)
	case io.Reader:
		_, _ = io.Copy(w, value)
	default:
		_, _ = io.WriteString(w, lang.ToString(value))
	}
}

func nilIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
