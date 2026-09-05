package lang

// DirectFn1 returns the plain Go closure behind fn for one-argument
// calls when fn is such a closure, a MetaFn wrapping one, or a
// multi-arity fn whose one-argument method is one.
func DirectFn1(fn any) (FnFunc1, bool) {
	switch f := fn.(type) {
	case FnFunc1:
		return f, true
	case *MetaFn:
		return f.fn1, f.fn1 != nil
	case *MultiArityFn:
		return f.fn1, f.fn1 != nil
	}
	return nil, false
}

// DirectFn2 is DirectFn1 for two-argument calls.
func DirectFn2(fn any) (FnFunc2, bool) {
	switch f := fn.(type) {
	case FnFunc2:
		return f, true
	case *MetaFn:
		return f.fn2, f.fn2 != nil
	case *MultiArityFn:
		return f.fn2, f.fn2 != nil
	}
	return nil, false
}

// DirectFn3 is DirectFn1 for three-argument calls.
func DirectFn3(fn any) (FnFunc3, bool) {
	switch f := fn.(type) {
	case FnFunc3:
		return f, true
	case *MetaFn:
		return f.fn3, f.fn3 != nil
	case *MultiArityFn:
		return f.fn3, f.fn3 != nil
	}
	return nil, false
}

// DirectFn4 is DirectFn1 for four-argument calls.
func DirectFn4(fn any) (FnFunc4, bool) {
	switch f := fn.(type) {
	case FnFunc4:
		return f, true
	case *MetaFn:
		return f.fn4, f.fn4 != nil
	case *MultiArityFn:
		return f.fn4, f.fn4 != nil
	}
	return nil, false
}
