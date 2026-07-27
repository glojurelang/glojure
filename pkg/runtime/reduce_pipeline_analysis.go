package runtime

import "github.com/glojurelang/glojure/pkg/ast"

// These names are also recognized by the pre-AST constant-form fast path.
// AST pipeline analysis itself lives in compiler.AnalyzePipeline.
var purePipelineCallbacks = map[string]map[string]bool{
	"map": {
		"dec":      true,
		"identity": true,
		"inc":      true,
	},
	"filter": {
		"even?": true,
		"neg?":  true,
		"odd?":  true,
		"pos?":  true,
		"zero?": true,
	},
}

func int64ConstValue(node *ast.Node) (int64, bool) {
	if node == nil || node.Op != ast.OpConst {
		return 0, false
	}
	value, ok := node.Sub.(*ast.ConstNode).Value.(int64)
	return value, ok
}

func pipelineTransformKind(
	operator, callback string,
) ReducePipelineTransformKind {
	if operator == "map" {
		switch callback {
		case "inc":
			return ReducePipelineMapInc
		case "dec":
			return ReducePipelineMapDec
		default:
			return ReducePipelineMapIdentity
		}
	}
	switch callback {
	case "odd?":
		return ReducePipelineFilterOdd
	case "even?":
		return ReducePipelineFilterEven
	case "pos?":
		return ReducePipelineFilterPos
	case "neg?":
		return ReducePipelineFilterNeg
	default:
		return ReducePipelineFilterZero
	}
}
