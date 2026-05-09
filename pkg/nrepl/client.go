package nrepl

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Client connects to a running nREPL server.
type Client struct {
	conn    net.Conn
	br      byteReaderInterface
	mu      sync.Mutex
	session string
	ns      string
}

// Connect dials an nREPL server and clones a session.
func Connect(host string, port int) (*Client, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("nrepl: connect %s: %w", addr, err)
	}
	c := &Client{
		conn: conn,
		br:   newByteReader(conn),
		ns:   "user",
	}
	if err := c.clone(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// Close closes the session and connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != "" {
		c.send(map[string]interface{}{
			"op":      "close",
			"session": c.session,
		})
	}
	return c.conn.Close()
}

// NS returns the current namespace from the server.
func (c *Client) NS() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ns
}

// Eval sends code to the server for evaluation. It returns the
// printed value, the current namespace, any stdout output, and any
// error. Stdout output is accumulated from "out" messages.
func (c *Client) Eval(code string) (value, ns, out string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.send(map[string]interface{}{
		"op":      "eval",
		"code":    code,
		"session": c.session,
		"ns":      c.ns,
	})

	var outBuf strings.Builder
	for {
		resp, readErr := c.recv()
		if readErr != nil {
			return "", c.ns, outBuf.String(), readErr
		}

		if v, ok := resp["out"].(string); ok {
			outBuf.WriteString(v)
		}
		if v, ok := resp["value"].(string); ok {
			value = v
		}
		if v, ok := resp["ns"].(string); ok {
			c.ns = v
			ns = v
		}
		if v, ok := resp["ex"].(string); ok {
			err = fmt.Errorf("%s", v)
		}

		if statusDone(resp) {
			break
		}
	}

	if ns == "" {
		ns = c.ns
	}
	return value, ns, outBuf.String(), err
}

// CompletionEntry holds a completion candidate with optional metadata.
type CompletionEntry struct {
	Candidate string
	NS        string // namespace the symbol is defined in
	Type      string // "namespace", "function", etc.
}

// Completions requests completions for prefix in the given namespace.
func (c *Client) Completions(prefix, ns string) ([]CompletionEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ns == "" {
		ns = c.ns
	}

	c.send(map[string]interface{}{
		"op":      "completions",
		"prefix":  prefix,
		"ns":      ns,
		"session": c.session,
	})

	var result []CompletionEntry
	for {
		resp, err := c.recv()
		if err != nil {
			return nil, err
		}

		if comps, ok := resp["completions"].([]interface{}); ok {
			for _, comp := range comps {
				if m, ok := comp.(map[string]interface{}); ok {
					if candidate, ok := m["candidate"].(string); ok {
						entry := CompletionEntry{Candidate: candidate}
						if v, ok := m["ns"].(string); ok {
							entry.NS = v
						}
						if v, ok := m["type"].(string); ok {
							entry.Type = v
						}
						result = append(result, entry)
					}
				}
			}
		}

		if statusDone(resp) {
			break
		}
	}
	return result, nil
}

func (c *Client) clone() error {
	c.send(map[string]interface{}{
		"op": "clone",
		"id": "clone-1",
	})
	resp, err := c.recv()
	if err != nil {
		return fmt.Errorf("nrepl: clone: %w", err)
	}
	id, ok := resp["new-session"].(string)
	if !ok {
		return fmt.Errorf("nrepl: clone: no session id in response")
	}
	c.session = id
	return nil
}

func (c *Client) send(msg map[string]interface{}) error {
	data, err := BencodeEncode(msg)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}

func (c *Client) recv() (map[string]interface{}, error) {
	val, err := bencodeRead(c.br)
	if err != nil {
		return nil, err
	}
	msg, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("nrepl: expected dict, got %T", val)
	}
	return msg, nil
}

func statusDone(msg map[string]interface{}) bool {
	status, ok := msg["status"].([]interface{})
	if !ok {
		return false
	}
	for _, s := range status {
		if s == "done" {
			return true
		}
	}
	return false
}
