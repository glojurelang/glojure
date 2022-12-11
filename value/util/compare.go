package util

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/value"
)

func Compare(x, y interface{}) int {
	if x == y {
		return 0
	}
	if x == nil {
		return -1
	}

	switch x := x.(type) {
	case value.Comparer:
		return x.Compare(y)
	case string:
		return strings.Compare(x, y.(string))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *value.BigInt, *value.Ratio, *value.BigDecimal:
		// TODO: add Cmp method to number ops
		v, ok := value.AsInt(value.Sub(x, y))
		if !ok {
			panic(fmt.Sprintf("cannot compare %T and %T", x, y))
		}
		return v
	default:
		panic(fmt.Errorf("cannot compare %T to %T", x, y))
	}
}
