package repl

import (
	"io"

	"github.com/glojurelang/glojure/pkg/lang"
)

type options struct {
	stdin     io.Reader
	stdout    io.Writer
	namespace string
	env       lang.Environment
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
