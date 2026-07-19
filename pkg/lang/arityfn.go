package lang

import "fmt"

// ArityFn represents a function with multiple fixed arities and an optional
// variadic method. The fixed InvokeN methods keep common call sites off the
// variadic []any path while Invoke and ApplyTo preserve general IFn behavior.
type ArityFn struct {
	meta        IPersistentMap
	fixed       [5]IFn
	fixedOther  map[int]IFn
	maxFixed    int
	variadic    IFn
	minVariadic int
}

func NewArityFn(
	fn0, fn1, fn2, fn3, fn4 IFn,
	variadic IFn,
	minVariadic int,
) ArityFn {
	f := ArityFn{
		fixed:       [5]IFn{fn0, fn1, fn2, fn3, fn4},
		variadic:    variadic,
		minVariadic: minVariadic,
	}
	for arity, method := range f.fixed {
		if method != nil {
			f.maxFixed = arity
		}
	}
	return f
}

func NewArityFnMethods(
	fixed map[int]IFn,
	variadic IFn,
	minVariadic int,
) ArityFn {
	f := ArityFn{
		variadic:    variadic,
		minVariadic: minVariadic,
	}
	for arity, method := range fixed {
		if arity < len(f.fixed) {
			f.fixed[arity] = method
		} else {
			if f.fixedOther == nil {
				f.fixedOther = make(map[int]IFn)
			}
			f.fixedOther[arity] = method
		}
		if arity > f.maxFixed {
			f.maxFixed = arity
		}
	}
	return f
}

func (f ArityFn) fixedMethod(arity int) IFn {
	if arity < len(f.fixed) {
		return f.fixed[arity]
	}
	return f.fixedOther[arity]
}

func (f ArityFn) Invoke(args ...any) any {
	if method := f.fixedMethod(len(args)); method != nil {
		return method.Invoke(args...)
	}
	if f.variadic != nil && len(args) >= f.minVariadic {
		return f.variadic.Invoke(args...)
	}
	panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments (%d)", len(args))))
}

func (f ArityFn) Invoke0() any {
	if method := f.fixed[0]; method != nil {
		return Apply0(method)
	}
	return f.Invoke()
}

func (f ArityFn) Invoke1(a0 any) any {
	if method := f.fixed[1]; method != nil {
		return Apply1(method, a0)
	}
	return f.Invoke(a0)
}

func (f ArityFn) Invoke2(a0, a1 any) any {
	if method := f.fixed[2]; method != nil {
		return Apply2(method, a0, a1)
	}
	return f.Invoke(a0, a1)
}

func (f ArityFn) Invoke3(a0, a1, a2 any) any {
	if method := f.fixed[3]; method != nil {
		return Apply3(method, a0, a1, a2)
	}
	return f.Invoke(a0, a1, a2)
}

func (f ArityFn) Invoke4(a0, a1, a2, a3 any) any {
	if method := f.fixed[4]; method != nil {
		return Apply4(method, a0, a1, a2, a3)
	}
	return f.Invoke(a0, a1, a2, a3)
}

func (f ArityFn) ApplyTo(args ISeq) any {
	original := args
	limit := f.maxFixed + 1
	if f.variadic != nil && f.minVariadic+1 > limit {
		limit = f.minVariadic + 1
	}
	arity := BoundedLength(args, limit)

	if method := f.fixedMethod(arity); method != nil {
		return method.ApplyTo(args)
	}
	if f.variadic != nil && arity >= f.minVariadic {
		return f.variadic.ApplyTo(original)
	}
	panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments (%d)", arity)))
}

func (f ArityFn) Meta() IPersistentMap {
	return f.meta
}

func (f ArityFn) WithMeta(meta IPersistentMap) any {
	copy := f
	copy.meta = meta
	return copy
}

var (
	_ IFn           = ArityFn{}
	_ IObj          = ArityFn{}
	_ FixedArityFn0 = ArityFn{}
	_ FixedArityFn1 = ArityFn{}
	_ FixedArityFn2 = ArityFn{}
	_ FixedArityFn3 = ArityFn{}
	_ FixedArityFn4 = ArityFn{}
)
