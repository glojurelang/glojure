package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
)

// nativeCoreAdd preserves clojure.core/+'s public arities while giving
// reducers a fixed-arity entry point that avoids a variadic dispatch on every
// element.
type nativeCoreAdd struct{}

func (nativeCoreAdd) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 0:
		return int64(0)
	case 1:
		return lang.MustAsNumber(args[0])
	case 2:
		return lang.Numbers.Add(args[0], args[1])
	default:
		result := lang.Numbers.Add(args[0], args[1])
		for _, arg := range args[2:] {
			result = lang.Numbers.Add(result, arg)
		}
		return result
	}
}

func (nativeCoreAdd) Invoke2(a, b interface{}) interface{} {
	return lang.Numbers.Add(a, b)
}

func (nativeCoreAdd) ReduceInt64(a, b int64) int64 {
	return checkedInt64Add(a, b)
}

func (fn nativeCoreAdd) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return int64(0)
	}
	if reducible, ok := args.(lang.IReduce); ok {
		if counted, ok := args.(lang.Counted); ok {
			switch counted.Count() {
			case 0:
				return int64(0)
			case 1:
				return lang.MustAsNumber(args.First())
			default:
				return reducible.Reduce(fn)
			}
		}
	}
	first := args.First()
	args = args.Next()
	if args == nil {
		return lang.MustAsNumber(first)
	}
	result := fn.Invoke2(first, args.First())
	for args = args.Next(); args != nil; args = args.Next() {
		result = fn.Invoke2(result, args.First())
	}
	return result
}

type nativeCoreSubtract struct{}

func (nativeCoreSubtract) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 0:
		panic(lang.NewIllegalArgumentError("wrong number of arguments (0)"))
	case 1:
		return lang.Numbers.Multiply(int64(-1), args[0])
	case 2:
		return lang.Numbers.Minus(args[0], args[1])
	default:
		result := lang.Numbers.Minus(args[0], args[1])
		for _, arg := range args[2:] {
			result = lang.Numbers.Minus(result, arg)
		}
		return result
	}
}

func (nativeCoreSubtract) Invoke1(a interface{}) interface{} {
	return lang.Numbers.Multiply(int64(-1), a)
}

func (nativeCoreSubtract) Invoke2(a, b interface{}) interface{} {
	return lang.Numbers.Minus(a, b)
}

func (nativeCoreSubtract) ReduceInt64(a, b int64) int64 {
	return checkedInt64Sub(a, b)
}

func (fn nativeCoreSubtract) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return fn.Invoke()
	}
	first := args.First()
	args = args.Next()
	if args == nil {
		return fn.Invoke1(first)
	}
	result := fn.Invoke2(first, args.First())
	args = args.Next()
	if args == nil {
		return result
	}
	if reducible, ok := args.(lang.IReduceInit); ok {
		return reducible.ReduceInit(fn, result)
	}
	for ; args != nil; args = args.Next() {
		result = fn.Invoke2(result, args.First())
	}
	return result
}

// nativeCoreStr avoids constructing a persistent rest sequence and reflecting
// through strings.Builder methods for ordinary string concatenation. It
// preserves clojure.core/str's nil handling and ToString conversion.
type nativeCoreStr struct{}

func (nativeCoreStr) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 0:
		return ""
	case 1:
		return nativeStrValue(args[0])
	case 2:
		return nativeStr2(args[0], args[1])
	default:
		var builder strings.Builder
		for _, arg := range args {
			if arg != nil {
				builder.WriteString(lang.ToString(arg))
			}
		}
		return builder.String()
	}
}

func (nativeCoreStr) Invoke0() interface{} {
	return ""
}

func (nativeCoreStr) Invoke1(value interface{}) interface{} {
	return nativeStrValue(value)
}

func (nativeCoreStr) Invoke2(a, b interface{}) interface{} {
	return nativeStr2(a, b)
}

func (nativeCoreStr) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return ""
	}
	var builder strings.Builder
	for ; args != nil; args = args.Next() {
		if value := args.First(); value != nil {
			builder.WriteString(lang.ToString(value))
		}
	}
	return builder.String()
}

func nativeStrValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return lang.ToString(value)
}

func nativeStr2(a, b interface{}) string {
	if a == nil {
		return nativeStrValue(b)
	}
	if b == nil {
		return lang.ToString(a)
	}
	var builder strings.Builder
	builder.WriteString(lang.ToString(a))
	builder.WriteString(lang.ToString(b))
	return builder.String()
}

// nativeCoreDeref keeps the common reference path on the IDeref interface.
// The compiled Clojure implementation remains the fallback for futures and
// other values whose dereference operation is exposed through host interop.
type nativeCoreDeref struct {
	fallback lang.IFn
}

func (fn nativeCoreDeref) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 1:
		return fn.Invoke1(args[0])
	case 3:
		return fn.Invoke3(args[0], args[1], args[2])
	default:
		return fn.fallback.Invoke(args...)
	}
}

func (fn nativeCoreDeref) Invoke1(ref interface{}) interface{} {
	if deref, ok := ref.(lang.IDeref); ok {
		return deref.Deref()
	}
	return lang.Apply1(fn.fallback, ref)
}

func (fn nativeCoreDeref) Invoke3(ref, timeoutMS, timeoutValue interface{}) interface{} {
	if deref, ok := ref.(lang.IBlockingDeref); ok {
		return deref.DerefWithTimeout(lang.AsInt64(timeoutMS), timeoutValue)
	}
	return lang.Apply3(fn.fallback, ref, timeoutMS, timeoutValue)
}

func (fn nativeCoreDeref) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return fn.fallback.ApplyTo(nil)
	}
	ref := args.First()
	args = args.Next()
	if args == nil {
		return fn.Invoke1(ref)
	}
	timeoutMS := args.First()
	args = args.Next()
	if args == nil {
		return fn.fallback.Invoke(ref, timeoutMS)
	}
	timeoutValue := args.First()
	if args.Next() == nil {
		return fn.Invoke3(ref, timeoutMS, timeoutValue)
	}
	return fn.fallback.ApplyTo(lang.NewCons(ref, lang.NewCons(timeoutMS, args)))
}

type fixedAtomSwap0 interface {
	Swap0(lang.IFn) interface{}
}

type fixedAtomSwap1 interface {
	Swap1(lang.IFn, interface{}) interface{}
}

type fixedAtomSwap2 interface {
	Swap2(lang.IFn, interface{}, interface{}) interface{}
}

// nativeCoreSwap dispatches directly to IAtom and uses optional fixed-arity
// methods when available. This removes reflected bound-method calls and rest
// sequence construction from ordinary swap! calls.
type nativeCoreSwap struct {
	fallback lang.IFn
}

func (fn nativeCoreSwap) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 2:
		return fn.Invoke2(args[0], args[1])
	case 3:
		return fn.Invoke3(args[0], args[1], args[2])
	case 4:
		return fn.Invoke4(args[0], args[1], args[2], args[3])
	default:
		if len(args) < 2 {
			return fn.fallback.Invoke(args...)
		}
		atom, ok := args[0].(lang.IAtom)
		if !ok {
			return fn.fallback.Invoke(args...)
		}
		return atom.Swap(args[1].(lang.IFn), lang.NewList(args[2:]...))
	}
}

func (fn nativeCoreSwap) Invoke2(atom, f interface{}) interface{} {
	if fixed, ok := atom.(fixedAtomSwap0); ok {
		return fixed.Swap0(f.(lang.IFn))
	}
	if atom, ok := atom.(lang.IAtom); ok {
		return atom.Swap(f.(lang.IFn), nil)
	}
	return lang.Apply2(fn.fallback, atom, f)
}

func (fn nativeCoreSwap) Invoke3(atom, f, x interface{}) interface{} {
	if fixed, ok := atom.(fixedAtomSwap1); ok {
		return fixed.Swap1(f.(lang.IFn), x)
	}
	if atom, ok := atom.(lang.IAtom); ok {
		return atom.Swap(f.(lang.IFn), lang.NewList(x))
	}
	return lang.Apply3(fn.fallback, atom, f, x)
}

func (fn nativeCoreSwap) Invoke4(atom, f, x, y interface{}) interface{} {
	if fixed, ok := atom.(fixedAtomSwap2); ok {
		return fixed.Swap2(f.(lang.IFn), x, y)
	}
	if atom, ok := atom.(lang.IAtom); ok {
		return atom.Swap(f.(lang.IFn), lang.NewList(x, y))
	}
	return lang.Apply4(fn.fallback, atom, f, x, y)
}

func (fn nativeCoreSwap) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return fn.fallback.ApplyTo(nil)
	}
	atom := args.First()
	args = args.Next()
	if args == nil {
		return fn.fallback.Invoke(atom)
	}
	f := args.First()
	rest := args.Next()
	if rest == nil {
		return fn.Invoke2(atom, f)
	}
	x := rest.First()
	rest = rest.Next()
	if rest == nil {
		return fn.Invoke3(atom, f, x)
	}
	y := rest.First()
	rest = rest.Next()
	if rest == nil {
		return fn.Invoke4(atom, f, x, y)
	}
	if atom, ok := atom.(lang.IAtom); ok {
		return atom.Swap(f.(lang.IFn), lang.NewCons(x, lang.NewCons(y, rest)))
	}
	return fn.fallback.ApplyTo(lang.NewCons(atom, lang.NewCons(f, lang.NewCons(x, lang.NewCons(y, rest)))))
}

// nativeCoreApply keeps Clojure's public apply semantics while routing the
// common fixed-leading-argument cases directly to the target function.
type nativeCoreApply struct{}

func (fn nativeCoreApply) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 2:
		return fn.Invoke2(args[0], args[1])
	case 3:
		return fn.Invoke3(args[0], args[1], args[2])
	case 4:
		return fn.Invoke4(args[0], args[1], args[2], args[3])
	case 5:
		return fn.invoke5(args[0], args[1], args[2], args[3], args[4])
	default:
		if len(args) < 2 {
			panic(lang.NewIllegalArgumentError("apply requires at least 2 arguments"))
		}
		return applyWithLeading(args[0], args[1:len(args)-1], args[len(args)-1])
	}
}

func (nativeCoreApply) Invoke2(fn, args interface{}) interface{} {
	return lang.ApplySeq(fn, lang.Seq(args))
}

func (nativeCoreApply) Invoke3(fn, x, args interface{}) interface{} {
	tail := lang.Seq(args)
	if tail == nil {
		return lang.Apply1(fn, x)
	}
	second := tail.First()
	tail = tail.Next()
	if tail == nil {
		return lang.Apply2(fn, x, second)
	}
	return lang.ApplySeq(fn, lang.NewCons(x, lang.NewCons(second, tail)))
}

func (nativeCoreApply) Invoke4(fn, x, y, args interface{}) interface{} {
	tail := lang.Seq(args)
	if tail == nil {
		return lang.Apply2(fn, x, y)
	}
	third := tail.First()
	tail = tail.Next()
	if tail == nil {
		return lang.Apply3(fn, x, y, third)
	}
	return lang.ApplySeq(fn, lang.NewCons(x, lang.NewCons(y, lang.NewCons(third, tail))))
}

func (nativeCoreApply) invoke5(fn, x, y, z, args interface{}) interface{} {
	tail := lang.Seq(args)
	if tail == nil {
		return lang.Apply3(fn, x, y, z)
	}
	fourth := tail.First()
	tail = tail.Next()
	if tail == nil {
		return lang.Apply4(fn, x, y, z, fourth)
	}
	return lang.ApplySeq(
		fn,
		lang.NewCons(x, lang.NewCons(y, lang.NewCons(z, lang.NewCons(fourth, tail)))),
	)
}

func (fn nativeCoreApply) ApplyTo(args lang.ISeq) interface{} {
	values := make([]interface{}, 0, lang.BoundedLength(args, 6))
	for ; args != nil; args = args.Next() {
		values = append(values, args.First())
	}
	return fn.Invoke(values...)
}

func applyWithLeading(fn interface{}, leading []interface{}, tail interface{}) interface{} {
	args := lang.Seq(tail)
	for i := len(leading) - 1; i >= 0; i-- {
		args = lang.NewCons(leading[i], args)
	}
	return lang.ApplySeq(fn, args)
}

type nativeCoreUpdateIn struct {
	assoc *lang.Var
	apply *lang.Var
}

func (fn nativeCoreUpdateIn) Invoke(args ...interface{}) interface{} {
	if len(args) < 3 {
		panic(lang.NewIllegalArgumentError("update-in requires at least 3 arguments"))
	}
	return nativeUpdateIn(args[0], args[1], args[2], args[3:], fn.assoc.Get(), fn.apply.Get())
}

func (fn nativeCoreUpdateIn) Invoke3(m, keys, updateFn interface{}) interface{} {
	return nativeUpdateIn(m, keys, updateFn, nil, fn.assoc.Get(), fn.apply.Get())
}

func (fn nativeCoreUpdateIn) Invoke4(m, keys, updateFn, arg interface{}) interface{} {
	return nativeUpdateIn(m, keys, updateFn, []interface{}{arg}, fn.assoc.Get(), fn.apply.Get())
}

func (fn nativeCoreUpdateIn) ApplyTo(args lang.ISeq) interface{} {
	if lang.BoundedLength(args, 3) < 3 {
		return fn.Invoke()
	}
	m := args.First()
	args = args.Next()
	keys := args.First()
	args = args.Next()
	updateFn := args.First()
	args = args.Next()
	if args == nil {
		return fn.Invoke3(m, keys, updateFn)
	}
	first := args.First()
	args = args.Next()
	if args == nil {
		return fn.Invoke4(m, keys, updateFn, first)
	}
	rest := []interface{}{first}
	for ; args != nil; args = args.Next() {
		rest = append(rest, args.First())
	}
	return nativeUpdateIn(m, keys, updateFn, rest, fn.assoc.Get(), fn.apply.Get())
}

type nativeCoreRequire struct {
	fallback         lang.IFn
	load             *lang.Var
	loadedLibs       *lang.Var
	loadingVerbosely *lang.Var
	contains         *lang.Var
	conj             *lang.Var
}

func (fn nativeCoreRequire) Invoke(args ...interface{}) interface{} {
	if len(args) == 1 {
		return fn.Invoke1(args[0])
	}
	return fn.fallback.Invoke(args...)
}

func (fn nativeCoreRequire) Invoke1(arg interface{}) interface{} {
	lib, ok := arg.(*lang.Symbol)
	if !ok {
		return lang.Apply1(fn.fallback, arg)
	}
	loadedLibs, ok := fn.loadedLibs.Get().(*lang.Ref)
	if !ok {
		return lang.Apply1(fn.fallback, arg)
	}
	if lang.IsTruthy(lang.Apply2(fn.contains.Get(), loadedLibs.Deref(), lib)) {
		return nil
	}

	undefinedOnEntry := lang.FindNamespace(lib) == nil
	defer func() {
		if value := recover(); value != nil {
			if undefinedOnEntry {
				lang.RemoveNamespace(lib)
			}
			panic(value)
		}
	}()

	loadingVerbosely := fn.loadingVerbosely.Deref()
	if !lang.IsTruthy(loadingVerbosely) {
		loadingVerbosely = nil
	}
	lang.PushThreadBindings(lang.NewMap(fn.loadingVerbosely, loadingVerbosely))
	defer lang.PopThreadBindings()

	resource := "/" + strings.NewReplacer("-", "_", ".", "/").Replace(lib.Name())
	lang.Apply1(fn.load.Get(), resource)
	lang.LockingTransaction.RunInTransaction(lang.FnFunc0(func() interface{} {
		loadedLibs.Commute(fn.conj.Get().(lang.IFn), lang.NewList(lib))
		return nil
	}))
	return nil
}

func (fn nativeCoreRequire) ApplyTo(args lang.ISeq) interface{} {
	if args != nil && args.Next() == nil {
		return fn.Invoke1(args.First())
	}
	return fn.fallback.ApplyTo(args)
}

func nativeUpdateIn(
	m, keys, fn interface{},
	args []interface{},
	assocFn, applyFn interface{},
) interface{} {
	if vector, ok := keys.(lang.IPersistentVector); ok {
		return nativeUpdateInVector(m, vector, 0, fn, args, assocFn, applyFn)
	}
	return nativeUpdateInSeq(m, lang.Seq(keys), fn, args, assocFn, applyFn)
}

func nativeUpdateInVector(
	m interface{},
	keys lang.IPersistentVector,
	index int,
	fn interface{},
	args []interface{},
	assocFn, applyFn interface{},
) interface{} {
	var key interface{}
	if index < keys.Count() {
		key = keys.Nth(index)
	}
	current := lang.Get(m, key)
	if index+1 < keys.Count() {
		current = nativeUpdateInVector(current, keys, index+1, fn, args, assocFn, applyFn)
	} else {
		current = applyUpdateInFunction(applyFn, fn, current, args)
	}
	return lang.Apply3(assocFn, m, key, current)
}

func nativeUpdateInSeq(
	m interface{},
	keys lang.ISeq,
	fn interface{},
	args []interface{},
	assocFn, applyFn interface{},
) interface{} {
	var key interface{}
	var remaining lang.ISeq
	if keys != nil {
		key = keys.First()
		remaining = keys.Next()
	}
	current := lang.Get(m, key)
	if remaining != nil {
		current = nativeUpdateInSeq(current, remaining, fn, args, assocFn, applyFn)
	} else {
		current = applyUpdateInFunction(applyFn, fn, current, args)
	}
	return lang.Apply3(assocFn, m, key, current)
}

func applyUpdateInFunction(applyFn, fn, current interface{}, args []interface{}) interface{} {
	var argSeq lang.ISeq
	if len(args) != 0 {
		argSeq = lang.NewList(args...)
	}
	return lang.Apply3(applyFn, fn, current, argSeq)
}

func installNativeCoreFunctions(core *lang.Namespace) {
	if add := core.FindInternedVar(lang.NewSymbol("+")); add != nil {
		add.BindRoot(nativeCoreAdd{})
	}
	if subtract := core.FindInternedVar(lang.NewSymbol("-")); subtract != nil {
		subtract.BindRoot(nativeCoreSubtract{})
	}
	if str := core.FindInternedVar(lang.NewSymbol("str")); str != nil {
		str.BindRoot(nativeCoreStr{})
	}
	if deref := core.FindInternedVar(lang.NewSymbol("deref")); deref != nil {
		if fallback, ok := deref.Get().(lang.IFn); ok {
			deref.BindRoot(nativeCoreDeref{fallback: fallback})
		}
	}
	if swap := core.FindInternedVar(lang.NewSymbol("swap!")); swap != nil {
		if fallback, ok := swap.Get().(lang.IFn); ok {
			swap.BindRoot(nativeCoreSwap{fallback: fallback})
		}
	}
	if apply := core.FindInternedVar(lang.NewSymbol("apply")); apply != nil {
		apply.BindRoot(nativeCoreApply{})
	}
	if updateIn := core.FindInternedVar(lang.NewSymbol("update-in")); updateIn != nil {
		updateIn.BindRoot(nativeCoreUpdateIn{
			assoc: core.FindInternedVar(lang.NewSymbol("assoc")),
			apply: core.FindInternedVar(lang.NewSymbol("apply")),
		})
	}
	if require := core.FindInternedVar(lang.NewSymbol("require")); require != nil {
		if fallback, ok := require.Get().(lang.IFn); ok {
			require.BindRoot(nativeCoreRequire{
				fallback:         fallback,
				load:             core.FindInternedVar(lang.NewSymbol("load")),
				loadedLibs:       core.FindInternedVar(lang.NewSymbol("*loaded-libs*")),
				loadingVerbosely: core.FindInternedVar(lang.NewSymbol("*loading-verbosely*")),
				contains:         core.FindInternedVar(lang.NewSymbol("contains?")),
				conj:             core.FindInternedVar(lang.NewSymbol("conj")),
			})
		}
	}
	installFixedArityCoreFunction(core, "map", 2, lang.FnFunc2(nativeMapSeq))
	installFixedArityCoreFunction(core, "mapv", 2, lang.FnFunc2(nativeMapv))
	installFixedArityCoreFunction(core, "filter", 2, lang.FnFunc2(nativeFilterSeq))
	installFixedArityCoreFunction(core, "take", 2, lang.FnFunc2(nativeTakeSeq))
	if mod := core.FindInternedVar(lang.NewSymbol("mod")); mod != nil {
		mod.BindRoot(lang.FnFunc2(nativeMod))
	}
	recordDefaultCoreRoots(
		core,
		"*", "+", "dec", "even?", "filter", "identity", "inc", "map",
		"neg?", "odd?", "pos?", "range", "reduce", "zero?",
		"take",
	)
}

func installFixedArityCoreFunction(
	core *lang.Namespace,
	name string,
	arity int,
	method lang.IFn,
) {
	vr := core.FindInternedVar(lang.NewSymbol(name))
	if vr == nil || !vr.IsBound() {
		return
	}
	original, ok := vr.Get().(lang.IFn)
	if !ok {
		return
	}
	var fixed [5]lang.IFn
	fixed[arity] = method
	vr.BindRoot(lang.NewArityFn(
		fixed[0], fixed[1], fixed[2], fixed[3], fixed[4],
		original, 0,
	))
}

func nativeMapv(fn, coll interface{}) interface{} {
	initial := lang.NewVector().AsTransient()
	reducer := lang.FnFunc2(func(result, value interface{}) interface{} {
		transient := result.(lang.ITransientCollection)
		transient.Conj(lang.Apply1(fn, value))
		return transient
	})
	var result interface{} = initial
	if reducible, ok := coll.(lang.IReduceInit); ok {
		result = reducible.ReduceInit(reducer, initial)
	} else {
		for seq := lang.Seq(coll); seq != nil; seq = seq.Next() {
			result = reducer(result, seq.First())
		}
	}
	return result.(lang.ITransientCollection).Persistent()
}

func nativeMod(num, div interface{}) interface{} {
	if numerator, ok := num.(int64); ok {
		if divisor, ok := div.(int64); ok {
			remainder := checkedInt64Remainder(numerator, divisor)
			if remainder == 0 || numerator > 0 == (divisor > 0) {
				return boxInt64(remainder)
			}
			return boxInt64(checkedInt64Add(remainder, divisor))
		}
	}

	remainder := lang.Numbers.Remainder(num, div)
	if lang.Numbers.IsZero(remainder) ||
		lang.Numbers.IsPos(num) == lang.Numbers.IsPos(div) {
		return remainder
	}
	return lang.Numbers.Add(remainder, div)
}

func nativeMapSeq(fn, coll interface{}) interface{} {
	return lang.NewLazySeq(func() interface{} {
		seq := lang.Seq(coll)
		if seq == nil {
			return nil
		}
		if chunked, ok := seq.(lang.IChunkedSeq); ok {
			input := chunked.ChunkedFirst()
			values := make([]interface{}, input.Count())
			for i := range values {
				values[i] = lang.Apply1(fn, input.Nth(i))
			}
			return lang.NewChunkedCons(
				lang.NewSliceChunk(values),
				nativeMapSeq(fn, chunked.ChunkedMore()).(lang.ISeq),
			)
		}
		return lang.NewCons(
			lang.Apply1(fn, seq.First()),
			nativeMapSeq(fn, seq.More()),
		)
	})
}

func nativeFilterSeq(pred, coll interface{}) interface{} {
	return lang.NewLazySeq(func() interface{} {
		seq := lang.Seq(coll)
		if seq == nil {
			return nil
		}
		if chunked, ok := seq.(lang.IChunkedSeq); ok {
			input := chunked.ChunkedFirst()
			values := make([]interface{}, 0, input.Count())
			for i := 0; i < input.Count(); i++ {
				value := input.Nth(i)
				if lang.IsTruthy(lang.Apply1(pred, value)) {
					values = append(values, value)
				}
			}
			more := nativeFilterSeq(pred, chunked.ChunkedMore()).(lang.ISeq)
			if len(values) == 0 {
				return more
			}
			return lang.NewChunkedCons(lang.NewSliceChunk(values), more)
		}
		value := seq.First()
		more := nativeFilterSeq(pred, seq.More())
		if lang.IsTruthy(lang.Apply1(pred, value)) {
			return lang.NewCons(value, more)
		}
		return more
	})
}

func nativeTakeSeq(n, coll interface{}) interface{} {
	remaining := lang.LongCast(n)
	return lang.NewLazySeq(func() interface{} {
		if remaining <= 0 {
			return nil
		}
		seq := lang.Seq(coll)
		if seq == nil {
			return nil
		}
		return lang.NewCons(seq.First(), nativeTakeSeq(remaining-1, seq.More()))
	})
}
