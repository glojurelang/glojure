package runtime

import (
	"fmt"
	"math"

	"github.com/glojurelang/glojure/pkg/lang"
)

// int64StepTarget normalizes the two optional primitive reducer interfaces.
// It is built once when a transducer is applied, never inside the hot loop.
type int64StepTarget struct {
	stepper lang.Int64ReductionStepper
	reducer lang.Int64Reducer
}

func newInt64StepTarget(value any) (int64StepTarget, bool) {
	if stepper, ok := value.(lang.Int64ReductionStepper); ok {
		return int64StepTarget{stepper: stepper}, true
	}
	if reducer, ok := value.(lang.Int64Reducer); ok {
		return int64StepTarget{reducer: reducer}, true
	}
	return int64StepTarget{}, false
}

func (target int64StepTarget) step(
	accumulator, value int64,
) (int64, bool) {
	if target.stepper != nil {
		return target.stepper.ReduceInt64Step(accumulator, value)
	}
	return target.reducer.ReduceInt64(accumulator, value), false
}

type nativeMapTransducer struct {
	meta lang.IPersistentMap
	fn   any
}

// NewMapTransducer returns the ordinary one-argument map transducer with an
// optional primitive reducer path when its callback and downstream reducer
// expose compatible entry points.
func NewMapTransducer(fn any) lang.IFn {
	return &nativeMapTransducer{fn: fn}
}

func (transducer *nativeMapTransducer) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(nativeTransducerArityError(len(args)))
	}
	return transducer.Invoke1(args[0])
}

func (transducer *nativeMapTransducer) Invoke1(reducing any) any {
	fallback := &nativeMapReducer{
		fn:       transducer.fn,
		reducing: reducing,
	}
	mapper, mapperOK := transducer.fn.(lang.Int64UnaryFn)
	target, targetOK := newInt64StepTarget(reducing)
	if mapperOK && targetOK {
		return &nativeInt64MapReducer{
			nativeMapReducer: fallback,
			mapper:           mapper,
			target:           target,
		}
	}
	return fallback
}

func (transducer *nativeMapTransducer) ApplyTo(args lang.ISeq) any {
	values := nativeFixedSeqArgs(args, 1)
	return transducer.Invoke1(values[0])
}

func (transducer *nativeMapTransducer) Meta() lang.IPersistentMap {
	return transducer.meta
}

func (transducer *nativeMapTransducer) WithMeta(
	meta lang.IPersistentMap,
) any {
	copy := *transducer
	copy.meta = meta
	return &copy
}

func (*nativeMapTransducer) IsFnValue() {}

type nativeMapReducer struct {
	meta     lang.IPersistentMap
	fn       any
	reducing any
}

func (reducer *nativeMapReducer) Invoke(args ...any) any {
	switch len(args) {
	case 0:
		return reducer.Invoke0()
	case 1:
		return reducer.Invoke1(args[0])
	case 2:
		return reducer.Invoke2(args[0], args[1])
	default:
		mapped := lang.Apply(reducer.fn, args[1:])
		return lang.Apply2(reducer.reducing, args[0], mapped)
	}
}

func (reducer *nativeMapReducer) Invoke0() any {
	return lang.Apply0(reducer.reducing)
}

func (reducer *nativeMapReducer) Invoke1(result any) any {
	return lang.Apply1(reducer.reducing, result)
}

func (reducer *nativeMapReducer) Invoke2(result, input any) any {
	return lang.Apply2(
		reducer.reducing,
		result,
		lang.Apply1(reducer.fn, input),
	)
}

func (reducer *nativeMapReducer) ApplyTo(args lang.ISeq) any {
	return reducer.Invoke(nativeSeqArgs(args)...)
}

func (reducer *nativeMapReducer) Meta() lang.IPersistentMap {
	return reducer.meta
}

func (reducer *nativeMapReducer) WithMeta(meta lang.IPersistentMap) any {
	copy := *reducer
	copy.meta = meta
	return &copy
}

func (*nativeMapReducer) IsFnValue() {}

type nativeInt64MapReducer struct {
	*nativeMapReducer
	mapper lang.Int64UnaryFn
	target int64StepTarget
}

func (reducer *nativeInt64MapReducer) ReduceInt64Step(
	accumulator, value int64,
) (int64, bool) {
	return reducer.target.step(
		accumulator,
		reducer.mapper.InvokeInt64(value),
	)
}

type nativeFilterTransducer struct {
	meta      lang.IPersistentMap
	predicate any
}

// NewFilterTransducer returns the ordinary one-argument filter transducer with
// an optional primitive predicate path.
func NewFilterTransducer(predicate any) lang.IFn {
	return &nativeFilterTransducer{predicate: predicate}
}

func (transducer *nativeFilterTransducer) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(nativeTransducerArityError(len(args)))
	}
	return transducer.Invoke1(args[0])
}

func (transducer *nativeFilterTransducer) Invoke1(reducing any) any {
	fallback := &nativeFilterReducer{
		predicate: transducer.predicate,
		reducing:  reducing,
	}
	predicate, predicateOK :=
		transducer.predicate.(lang.Int64PredicateFn)
	target, targetOK := newInt64StepTarget(reducing)
	if predicateOK && targetOK {
		return &nativeInt64FilterReducer{
			nativeFilterReducer: fallback,
			predicate:           predicate,
			target:              target,
		}
	}
	return fallback
}

func (transducer *nativeFilterTransducer) ApplyTo(args lang.ISeq) any {
	values := nativeFixedSeqArgs(args, 1)
	return transducer.Invoke1(values[0])
}

func (transducer *nativeFilterTransducer) Meta() lang.IPersistentMap {
	return transducer.meta
}

func (transducer *nativeFilterTransducer) WithMeta(
	meta lang.IPersistentMap,
) any {
	copy := *transducer
	copy.meta = meta
	return &copy
}

func (*nativeFilterTransducer) IsFnValue() {}

type nativeFilterReducer struct {
	meta      lang.IPersistentMap
	predicate any
	reducing  any
}

func (reducer *nativeFilterReducer) Invoke(args ...any) any {
	switch len(args) {
	case 0:
		return reducer.Invoke0()
	case 1:
		return reducer.Invoke1(args[0])
	case 2:
		return reducer.Invoke2(args[0], args[1])
	default:
		panic(nativeTransducerArityError(len(args)))
	}
}

func (reducer *nativeFilterReducer) Invoke0() any {
	return lang.Apply0(reducer.reducing)
}

func (reducer *nativeFilterReducer) Invoke1(result any) any {
	return lang.Apply1(reducer.reducing, result)
}

func (reducer *nativeFilterReducer) Invoke2(result, input any) any {
	if lang.IsTruthy(lang.Apply1(reducer.predicate, input)) {
		return lang.Apply2(reducer.reducing, result, input)
	}
	return result
}

func (reducer *nativeFilterReducer) ApplyTo(args lang.ISeq) any {
	return reducer.Invoke(nativeSeqArgs(args)...)
}

func (reducer *nativeFilterReducer) Meta() lang.IPersistentMap {
	return reducer.meta
}

func (reducer *nativeFilterReducer) WithMeta(meta lang.IPersistentMap) any {
	copy := *reducer
	copy.meta = meta
	return &copy
}

func (*nativeFilterReducer) IsFnValue() {}

type nativeInt64FilterReducer struct {
	*nativeFilterReducer
	predicate lang.Int64PredicateFn
	target    int64StepTarget
}

func (reducer *nativeInt64FilterReducer) ReduceInt64Step(
	accumulator, value int64,
) (int64, bool) {
	if !reducer.predicate.InvokeInt64Predicate(value) {
		return accumulator, false
	}
	return reducer.target.step(accumulator, value)
}

type nativeTakeTransducer struct {
	meta  lang.IPersistentMap
	limit any
}

// NewTakeTransducer returns the stateful one-argument take transducer. The
// ordinary path retains Clojure's numeric tower; an int64 limit can expose the
// primitive early-termination path.
func NewTakeTransducer(limit any) lang.IFn {
	return &nativeTakeTransducer{limit: limit}
}

func (transducer *nativeTakeTransducer) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(nativeTransducerArityError(len(args)))
	}
	return transducer.Invoke1(args[0])
}

func (transducer *nativeTakeTransducer) Invoke1(reducing any) any {
	fallback := &nativeTakeReducer{
		reducing:  reducing,
		remaining: transducer.limit,
	}
	target, ok := newInt64StepTarget(reducing)
	limit, int64Limit := transducer.limit.(int64)
	if ok && int64Limit && limit != math.MinInt64 {
		return &nativeInt64TakeReducer{
			nativeTakeReducer: fallback,
			target:            target,
			remaining:         limit,
		}
	}
	return fallback
}

func (transducer *nativeTakeTransducer) ApplyTo(args lang.ISeq) any {
	values := nativeFixedSeqArgs(args, 1)
	return transducer.Invoke1(values[0])
}

func (transducer *nativeTakeTransducer) Meta() lang.IPersistentMap {
	return transducer.meta
}

func (transducer *nativeTakeTransducer) WithMeta(
	meta lang.IPersistentMap,
) any {
	copy := *transducer
	copy.meta = meta
	return &copy
}

func (*nativeTakeTransducer) IsFnValue() {}

type nativeTakeReducer struct {
	meta      lang.IPersistentMap
	reducing  any
	remaining any
}

func (reducer *nativeTakeReducer) Invoke(args ...any) any {
	switch len(args) {
	case 0:
		return reducer.Invoke0()
	case 1:
		return reducer.Invoke1(args[0])
	case 2:
		return reducer.Invoke2(args[0], args[1])
	default:
		panic(nativeTransducerArityError(len(args)))
	}
}

func (reducer *nativeTakeReducer) Invoke0() any {
	return lang.Apply0(reducer.reducing)
}

func (reducer *nativeTakeReducer) Invoke1(result any) any {
	return lang.Apply1(reducer.reducing, result)
}

func (reducer *nativeTakeReducer) Invoke2(result, input any) any {
	remaining := reducer.remaining
	reducer.remaining = lang.Numbers.Dec(remaining)
	if lang.Numbers.IsPos(remaining) {
		result = lang.Apply2(reducer.reducing, result, input)
	}
	if !lang.Numbers.IsPos(reducer.remaining) && !lang.IsReduced(result) {
		return lang.NewReduced(result)
	}
	return result
}

func (reducer *nativeTakeReducer) ApplyTo(args lang.ISeq) any {
	return reducer.Invoke(nativeSeqArgs(args)...)
}

func (reducer *nativeTakeReducer) Meta() lang.IPersistentMap {
	return reducer.meta
}

func (reducer *nativeTakeReducer) WithMeta(meta lang.IPersistentMap) any {
	copy := *reducer
	copy.meta = meta
	return &copy
}

func (*nativeTakeReducer) IsFnValue() {}

type nativeInt64TakeReducer struct {
	*nativeTakeReducer
	target    int64StepTarget
	remaining int64
}

func (reducer *nativeInt64TakeReducer) ReduceInt64Step(
	accumulator, value int64,
) (int64, bool) {
	remaining := reducer.remaining
	reducer.remaining = checkedInt64Add(remaining, -1)
	reducer.nativeTakeReducer.remaining = reducer.remaining
	if remaining <= 0 {
		return accumulator, true
	}
	result, reduced := reducer.target.step(accumulator, value)
	return result, reduced || reducer.remaining <= 0
}

func nativeTransducerArityError(arity int) error {
	return lang.NewIllegalArgumentError(fmt.Sprintf(
		"wrong number of arguments (%d)", arity,
	))
}

func nativeFixedSeqArgs(args lang.ISeq, arity int) []any {
	if lang.BoundedLength(args, arity+1) != arity {
		panic(nativeTransducerArityError(lang.BoundedLength(args, arity+1)))
	}
	result := make([]any, arity)
	for index := range result {
		result[index] = args.First()
		args = args.Next()
	}
	return result
}

func nativeSeqArgs(args lang.ISeq) []any {
	result := make([]any, 0, lang.BoundedLength(args, 8))
	for ; args != nil; args = args.Next() {
		result = append(result, args.First())
	}
	return result
}

var (
	_ lang.IFn                   = (*nativeMapTransducer)(nil)
	_ lang.FixedArityFn1         = (*nativeMapTransducer)(nil)
	_ lang.IFn                   = (*nativeMapReducer)(nil)
	_ lang.FixedArityFn0         = (*nativeMapReducer)(nil)
	_ lang.FixedArityFn1         = (*nativeMapReducer)(nil)
	_ lang.FixedArityFn2         = (*nativeMapReducer)(nil)
	_ lang.Int64ReductionStepper = (*nativeInt64MapReducer)(nil)
	_ lang.IFn                   = (*nativeFilterTransducer)(nil)
	_ lang.FixedArityFn1         = (*nativeFilterTransducer)(nil)
	_ lang.Int64ReductionStepper = (*nativeInt64FilterReducer)(nil)
	_ lang.IFn                   = (*nativeTakeTransducer)(nil)
	_ lang.FixedArityFn1         = (*nativeTakeTransducer)(nil)
	_ lang.Int64ReductionStepper = (*nativeInt64TakeReducer)(nil)
)
