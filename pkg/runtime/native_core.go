package runtime

import (
	"reflect"
	"regexp"
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
		return nativeStrAny(args[0])
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
	return nativeStrAny(value)
}

func (nativeCoreStr) Invoke2(a, b interface{}) interface{} {
	return nativeStr2(a, b)
}

func (nativeCoreStr) Invoke3(a, b, c interface{}) interface{} {
	return nativeStrValue(a) + nativeStrValue(b) + nativeStrValue(c)
}

func (nativeCoreStr) Invoke4(a, b, c, d interface{}) interface{} {
	return nativeStrValue(a) + nativeStrValue(b) +
		nativeStrValue(c) + nativeStrValue(d)
}

func (nativeCoreStr) Invoke5(a, b, c, d, e interface{}) interface{} {
	return nativeStrValue(a) + nativeStrValue(b) +
		nativeStrValue(c) + nativeStrValue(d) + nativeStrValue(e)
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

func nativeStrAny(value interface{}) interface{} {
	if value == nil {
		return ""
	}
	if _, ok := value.(string); ok {
		return value
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
	return lang.ToString(a) + lang.ToString(b)
}

type nativeCoreRegexMatch struct {
	full bool
}

func (fn nativeCoreRegexMatch) Invoke(args ...interface{}) interface{} {
	if len(args) != 2 {
		panic(lang.NewIllegalArgumentError("regex match expects 2 arguments"))
	}
	return fn.Invoke2(args[0], args[1])
}

func (fn nativeCoreRegexMatch) Invoke2(pattern, value interface{}) interface{} {
	re := pattern.(*regexp.Regexp)
	text := value.(string)
	indices := re.FindStringSubmatchIndex(text)
	if indices == nil ||
		(fn.full && (indices[0] != 0 || indices[1] != len(text))) {
		return nil
	}
	if re.NumSubexp() == 0 {
		return text[indices[0]:indices[1]]
	}
	groups := make([]interface{}, len(indices)/2)
	for i := range groups {
		start, end := indices[2*i], indices[2*i+1]
		if start >= 0 {
			groups[i] = text[start:end]
		}
	}
	return lang.NewVector(groups...)
}

func (fn nativeCoreRegexMatch) ApplyTo(args lang.ISeq) interface{} {
	return fn.Invoke(seqToSlice(args)...)
}

type nativeCoreGetIn struct{}

var nativeGetInNotFound = new(struct{ _ byte })

func (nativeCoreGetIn) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 2:
		return getIn(args[0], args[1], nil, false)
	case 3:
		return getIn(args[0], args[1], args[2], true)
	default:
		panic(lang.NewIllegalArgumentError("get-in expects 2 or 3 arguments"))
	}
}

func (nativeCoreGetIn) Invoke2(value, keys interface{}) interface{} {
	return getIn(value, keys, nil, false)
}

func (nativeCoreGetIn) Invoke3(value, keys, notFound interface{}) interface{} {
	return getIn(value, keys, notFound, true)
}

func (nativeCoreGetIn) ApplyTo(args lang.ISeq) interface{} {
	return nativeCoreGetIn{}.Invoke(seqToSlice(args)...)
}

func getIn(value, keys, notFound interface{}, distinguishMissing bool) interface{} {
	if indexed, ok := keys.(lang.Indexed); ok {
		for i := 0; i < indexed.Count(); i++ {
			var found bool
			value, found = getInKey(value, indexed.Nth(i), distinguishMissing)
			if !found {
				return notFound
			}
		}
		return value
	}
	for seq := lang.Seq(keys); seq != nil; seq = seq.Next() {
		var found bool
		value, found = getInKey(value, seq.First(), distinguishMissing)
		if !found {
			return notFound
		}
	}
	return value
}

func getInKey(value, key interface{}, distinguishMissing bool) (interface{}, bool) {
	if !distinguishMissing {
		return lang.Get(value, key), true
	}
	result := lang.GetDefault(value, key, nativeGetInNotFound)
	return result, result != nativeGetInNotFound
}

type nativeCoreAssoc struct{}

func (nativeCoreAssoc) Invoke(args ...interface{}) interface{} {
	if len(args) < 3 || len(args)%2 == 0 {
		panic(lang.NewIllegalArgumentError(
			"assoc expects a collection followed by key/value pairs",
		))
	}
	result := args[0]
	for i := 1; i < len(args); i += 2 {
		result = lang.Assoc(result, args[i], args[i+1])
	}
	return result
}

func (nativeCoreAssoc) Invoke3(coll, key, value interface{}) interface{} {
	return lang.Assoc(coll, key, value)
}

func (nativeCoreAssoc) Invoke5(
	coll, key1, value1, key2, value2 interface{},
) interface{} {
	return lang.Assoc(lang.Assoc(coll, key1, value1), key2, value2)
}

func (fn nativeCoreAssoc) ApplyTo(args lang.ISeq) interface{} {
	if args == nil {
		return fn.Invoke()
	}
	result := args.First()
	args = args.Next()
	pairs := 0
	for args != nil {
		key := args.First()
		args = args.Next()
		if args == nil {
			panic(lang.NewIllegalArgumentError(
				"assoc expects even number of arguments after map/vector, found odd number",
			))
		}
		result = lang.Assoc(result, key, args.First())
		pairs++
		args = args.Next()
	}
	if pairs == 0 {
		return fn.Invoke(result)
	}
	return result
}

type nativeStringIncludes struct{}

func (nativeStringIncludes) Invoke(args ...interface{}) interface{} {
	if len(args) != 2 {
		panic(lang.NewIllegalArgumentError("includes? expects 2 arguments"))
	}
	return nativeStringIncludes{}.Invoke2(args[0], args[1])
}

func (nativeStringIncludes) Invoke2(value, substring interface{}) interface{} {
	return strings.Contains(lang.ToString(value), substring.(string))
}

func (nativeStringIncludes) ApplyTo(args lang.ISeq) interface{} {
	return nativeStringIncludes{}.Invoke(seqToSlice(args)...)
}

type nativeStringReplace struct {
	fallback lang.IFn
}

func (fn nativeStringReplace) Invoke(args ...interface{}) interface{} {
	if len(args) != 3 {
		return fn.fallback.Invoke(args...)
	}
	return fn.Invoke3(args[0], args[1], args[2])
}

func (fn nativeStringReplace) Invoke3(value, match, replacement interface{}) interface{} {
	if value == nil {
		panic(lang.NewIllegalArgumentError("cannot call clojure.string function on nil"))
	}
	text := lang.ToString(value)
	switch match := match.(type) {
	case string:
		if replacement, ok := replacement.(string); ok {
			return strings.ReplaceAll(text, match, replacement)
		}
	case lang.Char:
		switch replacement := replacement.(type) {
		case string:
			return strings.ReplaceAll(text, string(match), replacement)
		case lang.Char:
			return strings.ReplaceAll(text, string(match), string(replacement))
		}
	case *regexp.Regexp:
		if replacement, ok := replacement.(string); ok {
			return match.ReplaceAllString(text, replacement)
		}
	}
	return lang.Apply3(fn.fallback, value, match, replacement)
}

func (fn nativeStringReplace) ApplyTo(args lang.ISeq) interface{} {
	return fn.Invoke(seqToSlice(args)...)
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

// nativeCoreReduce dispatches directly to Glojure's reduction interfaces.
// The compiled Clojure function remains the fallback for protocol extensions
// on values that do not implement those interfaces.
type nativeCoreReduce struct {
	fallback lang.IFn
}

func (r nativeCoreReduce) Invoke(args ...interface{}) interface{} {
	switch len(args) {
	case 2:
		return r.Invoke2(args[0], args[1])
	case 3:
		return r.Invoke3(args[0], args[1], args[2])
	default:
		return r.fallback.Invoke(args...)
	}
}

func (r nativeCoreReduce) Invoke2(fn, coll interface{}) interface{} {
	if reducible, ok := coll.(lang.IReduce); ok {
		return reducible.Reduce(fn.(lang.IFn))
	}
	if values, ok := nativeArrayValue(coll); ok {
		reducer := fn.(lang.IFn)
		if values.Len() == 0 {
			return reducer.Invoke()
		}
		result := values.Index(0).Interface()
		return reduceNativeArray(reducer, result, values, 1)
	}
	return lang.Apply2(r.fallback, fn, coll)
}

func (r nativeCoreReduce) Invoke3(fn, initial, coll interface{}) interface{} {
	if reducible, ok := coll.(lang.IReduceInit); ok {
		return reducible.ReduceInit(fn.(lang.IFn), initial)
	}
	if values, ok := nativeArrayValue(coll); ok {
		return reduceNativeArray(fn.(lang.IFn), initial, values, 0)
	}
	return lang.Apply3(r.fallback, fn, initial, coll)
}

func (r nativeCoreReduce) ApplyTo(args lang.ISeq) interface{} {
	return r.Invoke(seqToSlice(args)...)
}

func nativeArrayValue(value interface{}) (reflect.Value, bool) {
	if value == nil {
		return reflect.Value{}, false
	}
	values := reflect.ValueOf(value)
	switch values.Kind() {
	case reflect.Array, reflect.Slice:
		return values, true
	default:
		return reflect.Value{}, false
	}
}

func reduceNativeArray(
	reducer lang.IFn,
	result interface{},
	values reflect.Value,
	start int,
) interface{} {
	for i := start; i < values.Len(); i++ {
		result = lang.Apply2(reducer, result, values.Index(i).Interface())
		if lang.IsReduced(result) {
			return result.(lang.IDeref).Deref()
		}
	}
	return result
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
	if reFind := core.FindInternedVar(lang.NewSymbol("re-find")); reFind != nil {
		reFind.BindRoot(nativeCoreRegexMatch{})
	}
	if reMatches := core.FindInternedVar(lang.NewSymbol("re-matches")); reMatches != nil {
		reMatches.BindRoot(nativeCoreRegexMatch{full: true})
	}
	if getIn := core.FindInternedVar(lang.NewSymbol("get-in")); getIn != nil {
		getIn.BindRoot(nativeCoreGetIn{})
	}
	if assoc := core.FindInternedVar(lang.NewSymbol("assoc")); assoc != nil {
		assoc.BindRoot(nativeCoreAssoc{})
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
	if reduce := core.FindInternedVar(lang.NewSymbol("reduce")); reduce != nil {
		if fallback, ok := reduce.Get().(lang.IFn); ok {
			reduce.BindRoot(nativeCoreReduce{fallback: fallback})
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
}

func recordOptimizableCoreRoots(core *lang.Namespace) {
	recordDefaultCoreRoots(core,
		"*", "+", "assoc", "atom", "cons", "conj", "count", "dec", "deref",
		"empty?", "even?", "filter", "first", "get", "identity", "inc", "map",
		"neg?", "next", "nth", "odd?", "peek", "pop", "pos?", "range",
		"reduce", "reset!", "seq", "swap!", "take", "zero?",
	)
}

func init() {
	if installNativeCoreOverrides {
		registerNativeNamespaceInitializer("clojure/string", installNativeStringFunctions)
	}
}

func installNativeStringFunctions() {
	namespace := lang.FindNamespace(lang.NewSymbol("clojure.string"))
	if namespace == nil {
		return
	}
	if includes := namespace.FindInternedVar(lang.NewSymbol("includes?")); includes != nil {
		includes.BindRoot(nativeStringIncludes{})
	}
	if replace := namespace.FindInternedVar(lang.NewSymbol("replace")); replace != nil {
		if fallback, ok := replace.Get().(lang.IFn); ok {
			replace.BindRoot(nativeStringReplace{fallback: fallback})
		}
	}
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
	if indexed, ok := coll.(lang.Indexed); ok {
		values := make([]interface{}, indexed.Count())
		for i := range values {
			values[i] = lang.Apply1(fn, indexed.Nth(i))
		}
		return lang.NewVector(values...)
	}

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
	if seq, ok := coll.(lang.ISeq); ok {
		_, lazy := seq.(*lang.LazySeq)
		_, chunked := seq.(lang.IChunkedSeq)
		if !lazy && !chunked {
			if source := seq.Seq(); source != nil {
				return lang.NewMappedSeq(fn, source)
			}
		}
	}
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
