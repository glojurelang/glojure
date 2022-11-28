package value

import (
	"fmt"
	"strconv"
)

func ToString(v interface{}) string {
	// if v is a Stringer, use its String method
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	switch v := v.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d.0", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	}

	return fmt.Sprintf("%T", v)
}
