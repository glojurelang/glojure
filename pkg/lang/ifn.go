package lang

// FnFunc is a wrapped Go function that implements the IFn interface.
type FnFunc func(args ...any) any

var (
	_ IFn = FnFunc(nil)
)

func NewFnFunc(fn func(args ...any) any) FnFunc {
	return FnFunc(fn)
}

func (f FnFunc) Invoke(args ...any) any {
	return f(args...)
}

func (f FnFunc) ApplyTo(args ISeq) any {
	return f(seqToSlice(args)...)
}

func (f FnFunc) Meta() IPersistentMap {
	return nil
}

func (f FnFunc) WithMeta(meta IPersistentMap) any {
	// no-op
	return f
}
