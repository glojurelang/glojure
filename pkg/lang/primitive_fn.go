package lang

import "fmt"

// Int64UnaryFnAdapter keeps an ordinary one-argument Clojure function and an
// inferred primitive entry point on the same value. Dynamic calls preserve the
// fallback's behavior; typed higher-order consumers can avoid boxing.
type Int64UnaryFnAdapter struct {
	meta     IPersistentMap
	fallback FnFunc1
	typed    func(int64) int64
}

func NewInt64UnaryFn(
	fallback FnFunc1,
	typed func(int64) int64,
) *Int64UnaryFnAdapter {
	return &Int64UnaryFnAdapter{
		fallback: fallback,
		typed:    typed,
	}
}

func (f *Int64UnaryFnAdapter) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments (%d)", len(args),
		)))
	}
	return f.Invoke1(args[0])
}

func (f *Int64UnaryFnAdapter) Invoke1(value any) any {
	return f.fallback(value)
}

func (f *Int64UnaryFnAdapter) InvokeInt64(value int64) int64 {
	return f.typed(value)
}

func (f *Int64UnaryFnAdapter) ApplyTo(args ISeq) any {
	values := requireFixedSeqArity(args, 1)
	return f.Invoke1(values[0])
}

func (f *Int64UnaryFnAdapter) Meta() IPersistentMap {
	return f.meta
}

func (f *Int64UnaryFnAdapter) WithMeta(meta IPersistentMap) any {
	copy := *f
	copy.meta = meta
	return &copy
}

func (*Int64UnaryFnAdapter) IsFnValue() {}

// Int64PredicateFnAdapter is the boolean-result counterpart of
// Int64UnaryFnAdapter.
type Int64PredicateFnAdapter struct {
	meta     IPersistentMap
	fallback FnFunc1
	typed    func(int64) bool
}

func NewInt64PredicateFn(
	fallback FnFunc1,
	typed func(int64) bool,
) *Int64PredicateFnAdapter {
	return &Int64PredicateFnAdapter{
		fallback: fallback,
		typed:    typed,
	}
}

func (f *Int64PredicateFnAdapter) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments (%d)", len(args),
		)))
	}
	return f.Invoke1(args[0])
}

func (f *Int64PredicateFnAdapter) Invoke1(value any) any {
	return f.fallback(value)
}

func (f *Int64PredicateFnAdapter) InvokeInt64Predicate(value int64) bool {
	return f.typed(value)
}

func (f *Int64PredicateFnAdapter) ApplyTo(args ISeq) any {
	values := requireFixedSeqArity(args, 1)
	return f.Invoke1(values[0])
}

func (f *Int64PredicateFnAdapter) Meta() IPersistentMap {
	return f.meta
}

func (f *Int64PredicateFnAdapter) WithMeta(meta IPersistentMap) any {
	copy := *f
	copy.meta = meta
	return &copy
}

func (*Int64PredicateFnAdapter) IsFnValue() {}

var (
	_ IFn              = (*Int64UnaryFnAdapter)(nil)
	_ IObj             = (*Int64UnaryFnAdapter)(nil)
	_ FixedArityFn1    = (*Int64UnaryFnAdapter)(nil)
	_ Int64UnaryFn     = (*Int64UnaryFnAdapter)(nil)
	_ IFn              = (*Int64PredicateFnAdapter)(nil)
	_ IObj             = (*Int64PredicateFnAdapter)(nil)
	_ FixedArityFn1    = (*Int64PredicateFnAdapter)(nil)
	_ Int64PredicateFn = (*Int64PredicateFnAdapter)(nil)
)
