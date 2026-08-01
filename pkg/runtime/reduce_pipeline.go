package runtime

import "github.com/glojurelang/glojure/pkg/lang"

type ReducePipelineTransformKind uint8

const (
	ReducePipelineMapIdentity ReducePipelineTransformKind = iota
	ReducePipelineMapInc
	ReducePipelineMapDec
	ReducePipelineMapSquare
	ReducePipelineFilterOdd
	ReducePipelineFilterEven
	ReducePipelineFilterPos
	ReducePipelineFilterNeg
	ReducePipelineFilterZero
)

type int64PipelineReducer struct {
	transforms []ReducePipelineTransformKind
}

func (r *int64PipelineReducer) ReduceInt64(result, value int64) int64 {
	// Keep this no-take loop direct. Routing it through the boolean-returning
	// helper below measurably slows the common full-reduction path.
	for _, transform := range r.transforms {
		switch transform {
		case ReducePipelineMapIdentity:
		case ReducePipelineMapInc:
			value = lang.CheckedAddInt64(value, 1)
		case ReducePipelineMapDec:
			value = lang.CheckedSubInt64(value, 1)
		case ReducePipelineMapSquare:
			value = lang.CheckedMultiplyInt64(value, value)
		case ReducePipelineFilterOdd:
			if value&1 == 0 {
				return result
			}
		case ReducePipelineFilterEven:
			if value&1 != 0 {
				return result
			}
		case ReducePipelineFilterPos:
			if value <= 0 {
				return result
			}
		case ReducePipelineFilterNeg:
			if value >= 0 {
				return result
			}
		case ReducePipelineFilterZero:
			if value != 0 {
				return result
			}
		default:
			panic(lang.NewIllegalArgumentError("unknown int64 reduce pipeline transform"))
		}
	}
	return lang.CheckedAddInt64(result, value)
}

func (r *int64PipelineReducer) reduceValue(
	result, value int64,
) (resultAfter int64, included bool) {
	for _, transform := range r.transforms {
		switch transform {
		case ReducePipelineMapIdentity:
		case ReducePipelineMapInc:
			value = lang.CheckedAddInt64(value, 1)
		case ReducePipelineMapDec:
			value = lang.CheckedSubInt64(value, 1)
		case ReducePipelineMapSquare:
			value = lang.CheckedMultiplyInt64(value, value)
		case ReducePipelineFilterOdd:
			if value&1 == 0 {
				return result, false
			}
		case ReducePipelineFilterEven:
			if value&1 != 0 {
				return result, false
			}
		case ReducePipelineFilterPos:
			if value <= 0 {
				return result, false
			}
		case ReducePipelineFilterNeg:
			if value >= 0 {
				return result, false
			}
		case ReducePipelineFilterZero:
			if value != 0 {
				return result, false
			}
		default:
			panic(lang.NewIllegalArgumentError("unknown int64 reduce pipeline transform"))
		}
	}
	result = lang.CheckedAddInt64(result, value)
	return result, true
}

func (r *int64PipelineReducer) Invoke(args ...interface{}) interface{} {
	if len(args) != 2 {
		panic(lang.NewIllegalArgumentError("wrong number of arguments"))
	}
	return r.Invoke2(args[0], args[1])
}

func (r *int64PipelineReducer) Invoke2(result, value interface{}) interface{} {
	return r.ReduceInt64(result.(int64), value.(int64))
}

func (r *int64PipelineReducer) ApplyTo(args lang.ISeq) interface{} {
	if lang.BoundedLength(args, 3) != 2 {
		return r.Invoke()
	}
	return r.Invoke2(args.First(), args.Next().First())
}

type int64TakingPipelineReducer struct {
	*int64PipelineReducer
	limit int64
	taken int64
}

func (r *int64TakingPipelineReducer) ReduceInt64Step(
	result, value int64,
) (int64, bool) {
	result, included := r.reduceValue(result, value)
	if included {
		r.taken++
	}
	return result, r.taken >= r.limit
}

// ReduceInt64Pipeline applies an ephemeral, proven-int64 sequence pipeline
// directly while summing its source. The analyzer calls it only for a constant
// integer range after guarding every participating core Var.
func ReduceInt64Pipeline(
	initial int64,
	coll interface{},
	transforms []ReducePipelineTransformKind,
	takeLimit int64,
) interface{} {
	if takeLimit == 0 {
		return lang.BoxInt64(initial)
	}
	reducer := &int64PipelineReducer{
		transforms: transforms,
	}
	if takeLimit < 0 {
		if reducible, ok := coll.(lang.IReduceInit); ok {
			return reducible.ReduceInit(reducer, initial)
		}
		result := initial
		for seq := lang.Seq(coll); seq != nil; seq = seq.Next() {
			result = reducer.ReduceInt64(result, seq.First().(int64))
		}
		return lang.BoxInt64(result)
	}

	stepper := &int64TakingPipelineReducer{
		int64PipelineReducer: reducer,
		limit:                takeLimit,
	}
	if reducible, ok := coll.(lang.Int64StepReducible); ok {
		return lang.BoxInt64(reducible.ReduceInt64Steps(stepper, initial))
	}
	result := initial
	for seq := lang.Seq(coll); seq != nil; seq = seq.Next() {
		var reduced bool
		result, reduced = stepper.ReduceInt64Step(result, seq.First().(int64))
		if reduced {
			break
		}
	}
	return lang.BoxInt64(result)
}
