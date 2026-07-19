package lang

import "fmt"

// FixedArityFnN are optional fast-call interfaces. IFn implementations can
// provide them to avoid constructing a variadic argument slice at hot call
// sites while retaining Invoke for general application.
type FixedArityFn0 interface{ Invoke0() any }
type FixedArityFn1 interface{ Invoke1(any) any }
type FixedArityFn2 interface{ Invoke2(any, any) any }
type FixedArityFn3 interface{ Invoke3(any, any, any) any }
type FixedArityFn4 interface{ Invoke4(any, any, any, any) any }

// FnFunc is a wrapped Go function that implements the IFn interface.
type FnFunc func(args ...any) any

var (
	_ IFn = FnFunc(nil)
	_ IFn = FnFunc0(nil)
	_ IFn = FnFunc1(nil)
	_ IFn = FnFunc2(nil)
	_ IFn = FnFunc3(nil)
	_ IFn = FnFunc4(nil)
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

// FnFunc0 is a zero-argument function implementing IFn with no []any allocation.
type FnFunc0 func() any

func NewFnFunc0(fn func() any) FnFunc0 { return FnFunc0(fn) }

func (f FnFunc0) Invoke(args ...any) any {
	if len(args) != 0 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 0, got %d", len(args))))
	}
	return f()
}

func (f FnFunc0) Invoke0() any { return f() }

func (f FnFunc0) ApplyTo(args ISeq) any {
	return f()
}

func (f FnFunc0) Meta() IPersistentMap          { return nil }
func (f FnFunc0) WithMeta(_ IPersistentMap) any { return f }

// FnFunc1 is a one-argument function implementing IFn with no []any allocation.
type FnFunc1 func(any) any

func NewFnFunc1(fn func(any) any) FnFunc1 { return FnFunc1(fn) }

func (f FnFunc1) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 1, got %d", len(args))))
	}
	return f(args[0])
}

func (f FnFunc1) Invoke1(a0 any) any { return f(a0) }

func (f FnFunc1) ApplyTo(args ISeq) any {
	return f.Invoke(seqToSlice(args)...)
}

func (f FnFunc1) Meta() IPersistentMap          { return nil }
func (f FnFunc1) WithMeta(_ IPersistentMap) any { return f }

// FnFunc2 is a two-argument function implementing IFn with no []any allocation.
type FnFunc2 func(any, any) any

func NewFnFunc2(fn func(any, any) any) FnFunc2 { return FnFunc2(fn) }

func (f FnFunc2) Invoke(args ...any) any {
	if len(args) != 2 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 2, got %d", len(args))))
	}
	return f(args[0], args[1])
}

func (f FnFunc2) Invoke2(a0, a1 any) any { return f(a0, a1) }

func (f FnFunc2) ApplyTo(args ISeq) any {
	return f.Invoke(seqToSlice(args)...)
}

func (f FnFunc2) Meta() IPersistentMap          { return nil }
func (f FnFunc2) WithMeta(_ IPersistentMap) any { return f }

// FnFunc3 is a three-argument function implementing IFn with no []any allocation.
type FnFunc3 func(any, any, any) any

func NewFnFunc3(fn func(any, any, any) any) FnFunc3 { return FnFunc3(fn) }

func (f FnFunc3) Invoke(args ...any) any {
	if len(args) != 3 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 3, got %d", len(args))))
	}
	return f(args[0], args[1], args[2])
}

func (f FnFunc3) Invoke3(a0, a1, a2 any) any { return f(a0, a1, a2) }

func (f FnFunc3) ApplyTo(args ISeq) any {
	return f.Invoke(seqToSlice(args)...)
}

func (f FnFunc3) Meta() IPersistentMap          { return nil }
func (f FnFunc3) WithMeta(_ IPersistentMap) any { return f }

// FnFunc4 is a four-argument function implementing IFn with no []any allocation.
type FnFunc4 func(any, any, any, any) any

func NewFnFunc4(fn func(any, any, any, any) any) FnFunc4 { return FnFunc4(fn) }

func (f FnFunc4) Invoke(args ...any) any {
	if len(args) != 4 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 4, got %d", len(args))))
	}
	return f(args[0], args[1], args[2], args[3])
}

func (f FnFunc4) Invoke4(a0, a1, a2, a3 any) any { return f(a0, a1, a2, a3) }

func (f FnFunc4) ApplyTo(args ISeq) any {
	return f.Invoke(seqToSlice(args)...)
}

func (f FnFunc4) Meta() IPersistentMap          { return nil }
func (f FnFunc4) WithMeta(_ IPersistentMap) any { return f }
