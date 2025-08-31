package lang

// FnFunc is a wrapped Go function that implements the IFn interface.
type FnFunc struct {
	fn   func(args ...any) any
	meta IPersistentMap
}

var (
	_ IFn = FnFunc{}
)

func NewFnFunc(fn func(args ...any) any) FnFunc {
	return FnFunc{
		fn: fn,
	}
}

func (f FnFunc) Invoke(args ...any) any {
	return f.fn(args...)
}

func (f FnFunc) ApplyTo(args ISeq) any {
	return f.fn(seqToSlice(args)...)
}

func (f FnFunc) Meta() IPersistentMap {
	return f.meta
}

func (f FnFunc) WithMeta(meta IPersistentMap) any {
	if f.meta == meta {
		return f
	}

	cpy := f
	cpy.meta = meta
	return cpy
}
