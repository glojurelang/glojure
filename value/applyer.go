package value

// Continuation represents the continuation of a computation.
type Continuation func() (interface{}, Continuation, error)

// Applyer is a value that can be applied to a list of arguments.
type Applyer interface {
	// TODO: should args be a sequence rather than a slice? Or an
	// interface{} that can be coerced to a sequence?
	Apply(env Environment, args []interface{}) (interface{}, error)
}

// ApplyerFunc is a function that can be applied to a list of
// arguments.
type ApplyerFunc func(env Environment, args []interface{}) (interface{}, error)

func (f ApplyerFunc) Apply(env Environment, args []interface{}) (interface{}, error) {
	return f(env, args)
}

// ContinuationApplyer is a value that can be applied to a list of
// arguments, possibly returning a continuation.
type ContinuationApplyer interface {
	ContinuationApply(env Environment, args []interface{}) (interface{}, Continuation, error)
}
