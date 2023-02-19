package value

// IFnFunc is a function that can be applied to a list of
// arguments.
type IFnFunc func(args ...interface{}) interface{}

var (
	_ IFn = IFnFunc(nil)
)

func (f IFnFunc) Invoke(args ...interface{}) interface{} {
	return f(args...)
}

func (f IFnFunc) ApplyTo(args ISeq) interface{} {
	return f(seqToSlice(args))
}
