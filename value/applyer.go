package value

// Continuation represents the continuation of a computation.
type Continuation func() (Value, Continuation, error)

// Applyer is a value that can be applied to a list of arguments.
type Applyer interface {
	Apply(env Environment, args []Value) (Value, error)
}

// ApplyerFunc is a function that can be applied to a list of
// arguments.
type ApplyerFunc func(env Environment, args []Value) (Value, error)

func (f ApplyerFunc) Apply(env Environment, args []Value) (Value, error) {
	return f(env, args)
}

// ContinuationApplyer is a value that can be applied to a list of
// arguments, possibly returning a continuation.
type ContinuationApplyer interface {
	ContinuationApply(env Environment, args []Value) (Value, Continuation, error)
}
