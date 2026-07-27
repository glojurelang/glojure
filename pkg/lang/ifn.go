package lang

//go:generate go run gen_fixed_arities.go

import "fmt"

// FnValue marks values created to represent Clojure functions. It is narrower
// than IFn: keywords and collections can be invoked, but are not functions.
type FnValue interface {
	IsFnValue()
}

// IsFn reports whether value represents a Clojure function.
func IsFn(value any) bool {
	_, ok := value.(FnValue)
	return ok
}

// FixedArityFnN are optional fast-call interfaces. IFn implementations can
// provide them to avoid constructing a variadic argument slice at hot call
// sites while retaining Invoke for general application.
type FixedArityFn0 interface{ Invoke0() any }
type FixedArityFn1 interface{ Invoke1(any) any }
type FixedArityFn2 interface{ Invoke2(any, any) any }
type FixedArityFn3 interface{ Invoke3(any, any, any) any }
type FixedArityFn4 interface{ Invoke4(any, any, any, any) any }
type FixedArityFn5 interface {
	Invoke5(any, any, any, any, any) any
}

// FnFunc is a wrapped Go function that implements the IFn interface.
type FnFunc func(args ...any) any

var (
	_ IFn = FnFunc(nil)
	_ IFn = VariadicFn{}
	_ IFn = FnFunc0(nil)
	_ IFn = FnFunc1(nil)
	_ IFn = FnFunc2(nil)
	_ IFn = FnFunc3(nil)
	_ IFn = FnFunc4(nil)
	_ IFn = FnFunc5(nil)
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

func (FnFunc) IsFnValue() {}

// VariadicFn keeps the ordinary variadic Go call path while allowing ApplyTo
// to pass an argument sequence to a Clojure variadic method without realizing
// its rest arguments.
type VariadicFn struct {
	meta          IPersistentMap
	requiredArity int
	doInvoke      func(fixed []any, rest ISeq) any
}

func NewVariadicFn(
	requiredArity int,
	doInvoke func(fixed []any, rest ISeq) any,
) VariadicFn {
	return VariadicFn{
		requiredArity: requiredArity,
		doInvoke:      doInvoke,
	}
}

func (f VariadicFn) Invoke(args ...any) any {
	if len(args) < f.requiredArity {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments: expected at least %d, got %d",
			f.requiredArity, len(args),
		)))
	}
	var rest ISeq
	if len(args) > f.requiredArity {
		rest = NewList(args[f.requiredArity:]...)
	}
	return f.doInvoke(args[:f.requiredArity], rest)
}

func (f VariadicFn) ApplyTo(args ISeq) any {
	if arity := BoundedLength(args, f.requiredArity); arity < f.requiredArity {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments: expected at least %d, got %d",
			f.requiredArity, arity,
		)))
	}
	fixed := make([]any, f.requiredArity)
	rest := args
	for i := range fixed {
		fixed[i] = rest.First()
		rest = rest.Next()
	}
	return f.doInvoke(fixed, rest)
}

func (f VariadicFn) Meta() IPersistentMap {
	return f.meta
}

func (f VariadicFn) WithMeta(meta IPersistentMap) any {
	copy := f
	copy.meta = meta
	return copy
}

func (VariadicFn) IsFnValue() {}

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
	requireFixedSeqArity(args, 0)
	return f()
}

func (f FnFunc0) Meta() IPersistentMap          { return nil }
func (f FnFunc0) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc0) IsFnValue()                      {}

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
	values := requireFixedSeqArity(args, 1)
	return f(values[0])
}

func (f FnFunc1) Meta() IPersistentMap          { return nil }
func (f FnFunc1) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc1) IsFnValue()                      {}

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
	values := requireFixedSeqArity(args, 2)
	return f(values[0], values[1])
}

func (f FnFunc2) Meta() IPersistentMap          { return nil }
func (f FnFunc2) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc2) IsFnValue()                      {}

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
	values := requireFixedSeqArity(args, 3)
	return f(values[0], values[1], values[2])
}

func (f FnFunc3) Meta() IPersistentMap          { return nil }
func (f FnFunc3) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc3) IsFnValue()                      {}

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
	values := requireFixedSeqArity(args, 4)
	return f(values[0], values[1], values[2], values[3])
}

func (f FnFunc4) Meta() IPersistentMap          { return nil }
func (f FnFunc4) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc4) IsFnValue()                      {}

// FnFunc5 is a five-argument function implementing IFn with no []any allocation.
type FnFunc5 func(any, any, any, any, any) any

func NewFnFunc5(fn func(any, any, any, any, any) any) FnFunc5 { return FnFunc5(fn) }

func (f FnFunc5) Invoke(args ...any) any {
	if len(args) != 5 {
		panic(NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments: expected 5, got %d", len(args))))
	}
	return f(args[0], args[1], args[2], args[3], args[4])
}

func (f FnFunc5) Invoke5(a0, a1, a2, a3, a4 any) any {
	return f(a0, a1, a2, a3, a4)
}

func (f FnFunc5) ApplyTo(args ISeq) any {
	values := requireFixedSeqArity(args, 5)
	return f(values[0], values[1], values[2], values[3], values[4])
}

func (f FnFunc5) Meta() IPersistentMap          { return nil }
func (f FnFunc5) WithMeta(_ IPersistentMap) any { return f }
func (FnFunc5) IsFnValue()                      {}

// requireFixedSeqArity reads up to five fixed arguments directly from an
// ISeq. Unlike seqToSlice, the successful path does not allocate a variadic
// argument slice. The full sequence is counted only on the exceptional path
// so that Invoke and ApplyTo report the same arity error.
func requireFixedSeqArity(args ISeq, expected int) [5]any {
	var values [5]any
	seq := args
	for i := 0; i < expected; i++ {
		if seq == nil {
			panic(NewIllegalArgumentError(fmt.Sprintf(
				"wrong number of arguments: expected %d, got %d",
				expected,
				i,
			)))
		}
		values[i] = seq.First()
		seq = seq.Next()
	}
	if seq != nil {
		got := expected
		for ; seq != nil; seq = seq.Next() {
			got++
		}
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments: expected %d, got %d",
			expected,
			got,
		)))
	}
	return values
}
