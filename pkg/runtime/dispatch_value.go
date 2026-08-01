package runtime

import "github.com/glojurelang/glojure/pkg/lang"

const (
	ExactDispatchNil uint8 = 1 << iota
	ExactDispatchKeyword
	ExactDispatchBool
	ExactDispatchString
	ExactDispatchInteger
)

// ExactDispatchValueSafe reports whether exact dispatch can compare value
// without invoking user-defined equality or hashing. Values outside the
// proved closed representation set retain ordinary multimethod selection.
func ExactDispatchValueSafe(value any, allowed uint8) bool {
	if value == nil {
		return allowed&ExactDispatchNil != 0
	}
	switch value.(type) {
	case lang.Keyword:
		return allowed&ExactDispatchKeyword != 0
	case bool:
		return allowed&ExactDispatchBool != 0
	case string:
		return allowed&ExactDispatchString != 0
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return allowed&ExactDispatchInteger != 0
	default:
		return false
	}
}
