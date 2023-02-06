package value

import (
	"context"
	"io"
)

type (
	// Environment is an interface for execution environments.
	Environment interface {
		// PushScope returns a new Environment with a scope nested inside
		// this environment's scope.
		//
		// TODO: make this work properly for lexical scoping.
		PushScope() Environment

		// WithRecurTarget returns a new Environment with the given recur
		// target. A recur form will return a RecurError with the given
		// target.
		WithRecurTarget(target interface{}) Environment

		// BindLocal binds the given name to the given value in the local
		// scope.
		BindLocal(sym *Symbol, v interface{})

		// DefVar defines a new var in the current namespace.
		DefVar(sym *Symbol, v interface{}) *Var

		// Eval evaluates a value representing an expression in this
		// environment.
		Eval(expr interface{}) (interface{}, error)

		EvalAST(n interface{}) (interface{}, error)

		// ResolveFile looks up a file in the environment. It should expand
		// relative paths to absolute paths. Relative paths are searched for
		// in the environments load paths.
		//
		// Deprecated
		ResolveFile(path string) (string, bool)

		// PushLoadPaths adds paths to the environment's list of load
		// paths. The provided paths will be searched for relative paths
		// first in the returned environment.
		//
		// Deprecated
		PushLoadPaths(paths []string) Environment

		// FindNamespace looks up a namespace in the environment. If the
		// namespace is not found, it returns nil.
		FindNamespace(sym *Symbol) *Namespace

		FindOrCreateNamespace(sym *Symbol) *Namespace

		SetCurrentNamespace(ns *Namespace)

		CurrentNamespace() *Namespace

		// Stdout returns the standard output stream for this environment.
		Stdout() io.Writer

		// Stderr returns the error output stream for this environment.
		Stderr() io.Writer

		// Context returns the context associated with this environment.
		Context() context.Context

		Errorf(form interface{}, format string, args ...interface{}) error
	}

	// RecurError is an error returned by a recur form.
	RecurError struct {
		Target interface{}
		Args   []interface{}
	}

	RecurTarget struct {
	}
)

func NewRecurTarget() *RecurTarget {
	return &RecurTarget{}
}

func (e *RecurError) Error() string {
	return "recur error (if you're seeing this, it's a bug)"
}

// Is returns true if the given error is a RecurError with the same
// target.
func (e *RecurError) Is(err error) bool {
	re, ok := err.(*RecurError)
	return ok && re.Target == e.Target
}
