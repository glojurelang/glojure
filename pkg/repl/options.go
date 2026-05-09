package repl

import (
	"io"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/nrepl"
)

type options struct {
	stdin       io.Reader
	stdout      io.Writer
	namespace   string
	env         lang.Environment
	nreplClient *nrepl.Client
	nreplServer *nrepl.Server
}

// Option is a functional option for the REPL.
type Option func(*options)

// WithStdin sets the stdin for the REPL.
func WithStdin(r io.Reader) Option {
	return func(o *options) {
		o.stdin = r
	}
}

// WithStdout sets the stdout for the REPL.
func WithStdout(w io.Writer) Option {
	return func(o *options) {
		o.stdout = w
	}
}

// WithEnvironment sets the execution environment for the REPL.
func WithEnvironment(env lang.Environment) Option {
	return func(o *options) {
		o.env = env
	}
}

// WithNREPLClient configures the REPL to send code to a remote
// nREPL server instead of evaluating locally.
func WithNREPLClient(c *nrepl.Client) Option {
	return func(o *options) {
		o.nreplClient = c
	}
}
