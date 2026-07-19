package runtime

import "github.com/glojurelang/glojure/pkg/lang"

type ReducePipelineTransformKind uint8

const (
	ReducePipelineMapIdentity ReducePipelineTransformKind = iota
	ReducePipelineMapInc
	ReducePipelineMapDec
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
	for _, transform := range r.transforms {
		switch transform {
		case ReducePipelineMapIdentity:
		case ReducePipelineMapInc:
			value = lang.CheckedAddInt64(value, 1)
		case ReducePipelineMapDec:
			value = lang.CheckedSubInt64(value, 1)
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

// ReduceInt64Pipeline applies an ephemeral, proven-int64 sequence pipeline
// directly while summing its source. Generated AOT code calls it only for a
// constant integer range after guarding every participating core Var.
func ReduceInt64Pipeline(
	initial int64,
	coll interface{},
	transforms []ReducePipelineTransformKind,
) interface{} {
	step := &int64PipelineReducer{transforms: transforms}
	if reducible, ok := coll.(lang.IReduceInit); ok {
		return reducible.ReduceInit(step, initial)
	}
	var result interface{} = initial
	for seq := lang.Seq(coll); seq != nil; seq = seq.Next() {
		result = step.Invoke2(result, seq.First())
	}
	return result
}
