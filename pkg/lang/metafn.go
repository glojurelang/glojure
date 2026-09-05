package lang

// MetaFn attaches metadata to a function value whose own type cannot
// carry any, such as the FnFuncN closure types that compiled single
// arity fns become. Calls forward to the wrapped function, using the
// fixed arity fast paths when the wrapped value supports them.
type MetaFn struct {
	meta IPersistentMap
	fn   IFn
}

var (
	_ IFn  = (*MetaFn)(nil)
	_ IObj = (*MetaFn)(nil)
)

// NewMetaFn wraps fn with meta. A nil meta returns fn unchanged.
func NewMetaFn(fn IFn, meta IPersistentMap) any {
	if meta == nil {
		return fn
	}
	return &MetaFn{meta: meta, fn: fn}
}

// Fn returns the wrapped function.
func (f *MetaFn) Fn() IFn { return f.fn }

func (f *MetaFn) Invoke(args ...any) any { return f.fn.Invoke(args...) }

func (f *MetaFn) ApplyTo(args ISeq) any { return f.fn.ApplyTo(args) }

func (f *MetaFn) Meta() IPersistentMap { return f.meta }

func (f *MetaFn) WithMeta(meta IPersistentMap) any {
	return NewMetaFn(f.fn, meta)
}

func (*MetaFn) IsFnValue() {}

func (f *MetaFn) Invoke0() any { return Apply0(f.fn) }

func (f *MetaFn) Invoke1(a0 any) any { return Apply1(f.fn, a0) }

func (f *MetaFn) Invoke2(a0, a1 any) any { return Apply2(f.fn, a0, a1) }

func (f *MetaFn) Invoke3(a0, a1, a2 any) any {
	return Apply3(f.fn, a0, a1, a2)
}

func (f *MetaFn) Invoke4(a0, a1, a2, a3 any) any {
	return Apply4(f.fn, a0, a1, a2, a3)
}

func (f *MetaFn) Invoke5(a0, a1, a2, a3, a4 any) any {
	return Apply5(f.fn, a0, a1, a2, a3, a4)
}
