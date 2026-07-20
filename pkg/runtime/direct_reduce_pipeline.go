package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
)

type directInt64ReducePipelinePlan struct {
	initial    int64
	rangeStart int64
	rangeEnd   int64
	rangeStep  int64
	transforms []ReducePipelineTransformKind
	takeLimit  int64
}

func (env *environment) evalDirectInt64ReducePipeline(
	form interface{},
	currentNS *lang.Namespace,
) (result interface{}, ok bool, err error) {
	plan, ok := analyzeDirectInt64ReducePipeline(form, currentNS)
	if !ok {
		return nil, false, nil
	}

	defer env.recoverDirectInvoke(form, &result, &err)
	result = ReduceInt64Pipeline(
		plan.initial,
		lang.NewLongRange(plan.rangeStart, plan.rangeEnd, plan.rangeStep),
		plan.transforms,
		plan.takeLimit,
	)
	return result, true, nil
}

func analyzeDirectInt64ReducePipeline(
	form interface{},
	currentNS *lang.Namespace,
) (*directInt64ReducePipelinePlan, bool) {
	seq, ok := form.(lang.ISeq)
	if !ok || seq == nil {
		return nil, false
	}
	operator, ok := seq.First().(*lang.Symbol)
	if !ok || !isDefaultCoreSymbol(currentNS, operator, "reduce") {
		return nil, false
	}
	args := directArgs(seq.Next())
	if len(args) != 3 {
		return nil, false
	}
	reducer, ok := args[0].(*lang.Symbol)
	if !ok || !isDefaultCoreSymbol(currentNS, reducer, "+") {
		return nil, false
	}
	initial, ok := args[1].(int64)
	if !ok {
		return nil, false
	}

	start, end, step, transforms, takeLimit, ok :=
		analyzeDirectPipelineSource(args[2], currentNS)
	if !ok || len(transforms) == 0 {
		return nil, false
	}
	return &directInt64ReducePipelinePlan{
		initial:    initial,
		rangeStart: start,
		rangeEnd:   end,
		rangeStep:  step,
		transforms: transforms,
		takeLimit:  takeLimit,
	}, true
}

func analyzeDirectPipelineSource(
	form interface{},
	currentNS *lang.Namespace,
) (
	start, end, step int64,
	transforms []ReducePipelineTransformKind,
	takeLimit int64,
	ok bool,
) {
	operator, args, ok := directCall(form)
	if !ok {
		return 0, 0, 0, nil, -1, false
	}
	if isDefaultCoreSymbol(currentNS, operator, "range") {
		start, end, step, ok = directInt64RangeArgs(args)
		return start, end, step, nil, -1, ok
	}
	if len(args) != 2 {
		return 0, 0, 0, nil, -1, false
	}

	if isDefaultCoreSymbol(currentNS, operator, "take") {
		limit, constant := args[0].(int64)
		if !constant || limit < 0 {
			return 0, 0, 0, nil, -1, false
		}
		start, end, step, transforms, innerLimit, ok :=
			analyzeDirectPipelineSource(args[1], currentNS)
		if !ok || innerLimit >= 0 {
			return 0, 0, 0, nil, -1, false
		}
		return start, end, step, transforms, limit, true
	}

	var operatorName string
	switch {
	case isDefaultCoreSymbol(currentNS, operator, "map"):
		operatorName = "map"
	case isDefaultCoreSymbol(currentNS, operator, "filter"):
		operatorName = "filter"
	default:
		return 0, 0, 0, nil, -1, false
	}
	kind, ok := directPipelineTransform(operatorName, args[0], currentNS)
	if !ok {
		return 0, 0, 0, nil, -1, false
	}

	start, end, step, transforms, takeLimit, ok =
		analyzeDirectPipelineSource(args[1], currentNS)
	// An inner take must count values before this outer transform, whereas
	// ReduceInt64Pipeline's limit counts the final transformed output.
	if !ok || takeLimit >= 0 {
		return 0, 0, 0, nil, -1, false
	}
	transforms = append(transforms, kind)
	return start, end, step, transforms, -1, true
}

func directInt64RangeArgs(args []interface{}) (start, end, step int64, ok bool) {
	switch len(args) {
	case 1:
		end, ok = args[0].(int64)
		return 0, end, 1, ok
	case 2:
		start, ok = args[0].(int64)
		if !ok {
			return 0, 0, 0, false
		}
		end, ok = args[1].(int64)
		return start, end, 1, ok
	case 3:
		start, ok = args[0].(int64)
		if !ok {
			return 0, 0, 0, false
		}
		end, ok = args[1].(int64)
		if !ok {
			return 0, 0, 0, false
		}
		step, ok = args[2].(int64)
		return start, end, step, ok
	default:
		return 0, 0, 0, false
	}
}

func directPipelineTransform(
	operator string,
	callback interface{},
	currentNS *lang.Namespace,
) (ReducePipelineTransformKind, bool) {
	if symbol, ok := callback.(*lang.Symbol); ok {
		name := symbol.Name()
		if purePipelineCallbacks[operator][name] &&
			isDefaultCoreSymbol(currentNS, symbol, name) {
			return pipelineTransformKind(operator, name), true
		}
	}
	if operator == "map" && isDirectInt64SquareFn(callback, currentNS) {
		return ReducePipelineMapSquare, true
	}
	return 0, false
}

func isDirectInt64SquareFn(
	form interface{},
	currentNS *lang.Namespace,
) bool {
	operator, args, ok := directCall(form)
	if !ok || operator.String() != "fn*" || len(args) != 2 {
		return false
	}
	params, ok := args[0].(lang.IPersistentVector)
	if !ok || params.Count() != 1 {
		return false
	}
	param, ok := params.Nth(0).(*lang.Symbol)
	if !ok {
		return false
	}
	multiply, bodyArgs, ok := directCall(args[1])
	if !ok || len(bodyArgs) != 2 ||
		!isDefaultCoreSymbol(currentNS, multiply, "*") {
		return false
	}
	left, leftOK := bodyArgs[0].(*lang.Symbol)
	right, rightOK := bodyArgs[1].(*lang.Symbol)
	return leftOK && rightOK &&
		left.Equals(param) && right.Equals(param)
}

func directCall(form interface{}) (*lang.Symbol, []interface{}, bool) {
	seq, ok := form.(lang.ISeq)
	if !ok || seq == nil {
		return nil, nil, false
	}
	operator, ok := seq.First().(*lang.Symbol)
	if !ok {
		return nil, nil, false
	}
	return operator, directArgs(seq.Next()), true
}

func directArgs(seq lang.ISeq) []interface{} {
	args := make([]interface{}, 0, 3)
	for ; seq != nil; seq = seq.Next() {
		args = append(args, seq.First())
	}
	return args
}

func isDefaultCoreSymbol(
	currentNS *lang.Namespace,
	symbol *lang.Symbol,
	name string,
) bool {
	if symbol.Name() != name {
		return false
	}
	vr := directInvokeVar(currentNS, symbol)
	return vr != nil &&
		vr.Namespace() == lang.NSCore &&
		vr.Symbol().String() == name &&
		IsDefaultCoreVar(vr)
}
