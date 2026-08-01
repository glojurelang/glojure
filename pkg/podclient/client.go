// Package podclient implements the native Babashka pod protocol for Glojure.
package podclient

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/nrepl"
	"github.com/glojurelang/glojure/pkg/reader"
)

// Client owns a pod subprocess and its dynamically installed namespaces.
type Client struct {
	command    *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	format     string
	operations map[string]bool
	namespaces []string
	id         string
	mu         sync.Mutex
	nextID     int64
}

// Start launches a local pod executable, describes it, and installs its vars
// into the global Glojure environment.
func Start(executable string, arguments ...string) (*Client, error) {
	command := exec.Command(executable, arguments...)
	command.Env = append(os.Environ(), "BABASHKA_POD=true")
	command.Stderr = os.Stderr
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	client := &Client{
		command:    command,
		stdin:      stdin,
		stdout:     stdout,
		operations: map[string]bool{},
	}
	reply, err := client.request(map[string]interface{}{
		"op": "describe",
		"id": "describe",
	})
	if err != nil {
		_ = command.Process.Kill()
		return nil, err
	}
	client.format, _ = reply["format"].(string)
	if client.format != "edn" {
		_ = command.Process.Kill()
		return nil, fmt.Errorf("podclient: payload format %q is not supported yet", client.format)
	}
	if operations, ok := reply["ops"].(map[string]interface{}); ok {
		for operation := range operations {
			client.operations[operation] = true
		}
	}
	namespaces, _ := reply["namespaces"].([]interface{})
	for _, rawNamespace := range namespaces {
		namespace, ok := rawNamespace.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := namespace["name"].(string)
		if name == "" {
			continue
		}
		if client.id == "" {
			client.id = name
		}
		client.namespaces = append(client.namespaces, name)
		ns := lang.FindOrCreateNamespace(lang.NewSymbol(name))
		vars, _ := namespace["vars"].([]interface{})
		for _, rawVar := range vars {
			variable, ok := rawVar.(map[string]interface{})
			if !ok {
				continue
			}
			varName, _ := variable["name"].(string)
			if varName == "" {
				continue
			}
			qualifiedName := name + "/" + varName
			fn := lang.NewFnFunc(func(args ...interface{}) interface{} {
				value, invokeErr := client.Invoke(qualifiedName, args)
				if invokeErr != nil {
					panic(invokeErr)
				}
				return value
			})
			ns.InternWithValue(lang.NewSymbol(varName), fn, true)
		}
	}
	if client.id == "" {
		client.id = fmt.Sprintf("pod-%d", command.Process.Pid)
	}
	return client, nil
}

// StartCommand launches a complete Clojure command collection.
func StartCommand(command interface{}) (*Client, error) {
	var arguments []string
	for values := lang.Seq(command); values != nil; values = values.Next() {
		arguments = append(arguments, lang.ToString(values.First()))
	}
	if len(arguments) == 0 {
		return nil, fmt.Errorf("podclient: command must not be empty")
	}
	return Start(arguments[0], arguments[1:]...)
}

// ID returns the pod identifier.
func (c *Client) ID() string { return c.id }

// Invoke calls one pod var synchronously.
func (c *Client) Invoke(variable string, arguments interface{}) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := fmt.Sprintf("%d", c.nextID)
	var argumentValues []interface{}
	for values := lang.Seq(arguments); values != nil; values = values.Next() {
		argumentValues = append(argumentValues, values.First())
	}
	reply, err := c.requestUnlocked(map[string]interface{}{
		"op":   "invoke",
		"id":   id,
		"var":  variable,
		"args": lang.PrintString(lang.NewVector(argumentValues...)),
	})
	if err != nil {
		return nil, err
	}
	if message, ok := reply["ex-message"].(string); ok {
		return nil, fmt.Errorf("%s", message)
	}
	value, _ := reply["value"].(string)
	return reader.New(strings.NewReader(value)).ReadOne()
}

// Close unloads namespaces and stops the pod subprocess.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.operations["shutdown"] {
		_, _ = c.requestUnlocked(map[string]interface{}{
			"op": "shutdown",
			"id": "shutdown",
		})
	} else if c.command.Process != nil {
		_ = c.command.Process.Kill()
	}
	for _, name := range c.namespaces {
		lang.RemoveNamespace(lang.NewSymbol(name))
	}
	_ = c.stdin.Close()
	_ = c.stdout.Close()
	return c.command.Wait()
}

func (c *Client) request(message map[string]interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requestUnlocked(message)
}

func (c *Client) requestUnlocked(message map[string]interface{}) (map[string]interface{}, error) {
	data, err := nrepl.BencodeEncode(message)
	if err != nil {
		return nil, err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return nil, err
	}
	raw, err := nrepl.BencodeDecode(c.stdout)
	if err != nil {
		return nil, err
	}
	reply, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("podclient: invalid response %T", raw)
	}
	return reply, nil
}
