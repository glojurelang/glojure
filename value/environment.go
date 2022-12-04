package value

import (
	"context"
	"io"
)

// Environment is an interface for execution environments.
type Environment interface {
	// PushScope returns a new Environment with a scope nested inside
	// this environment's scope.
	PushScope() Environment

	// Define defines a variable in the current scope.
	Define(name string, v interface{})

	// Eval evaluates a value representing an expression in this
	// environment.
	Eval(expr interface{}) (interface{}, error)

	// ResolveFile looks up a file in the environment. It should expand
	// relative paths to absolute paths. Relative paths are searched for
	// in the environments load paths.
	ResolveFile(path string) (string, bool)

	// PushLoadPaths adds paths to the environment's list of load
	// paths. The provided paths will be searched for relative paths
	// first in the returned environment.
	PushLoadPaths(paths []string) Environment

	// FindNamespace looks up a namespace in the environment. If the
	// namespace is not found, it returns nil.
	FindNamespace(name string) *Namespace

	// Stdout returns the standard output stream for this environment.
	Stdout() io.Writer

	// Stderr returns the error output stream for this environment.
	Stderr() io.Writer

	// Context returns the context associated with this environment.
	Context() context.Context
}
