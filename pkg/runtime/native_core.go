package runtime

import "github.com/glojurelang/glojure/pkg/lang"

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

func installNativeCoreFunctions(core *lang.Namespace) {
	if add := core.FindInternedVar(lang.NewSymbol("+")); add != nil {
		add.BindRoot(nativeCoreAdd{})
	}
	installFixedArityCoreFunction(core, "map", 2, lang.FnFunc2(nativeMapSeq))
	installFixedArityCoreFunction(core, "filter", 2, lang.FnFunc2(nativeFilterSeq))
	installFixedArityCoreFunction(core, "take", 2, lang.FnFunc2(nativeTakeSeq))
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
