package value

import "fmt"

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
	}

	return fmt.Sprintf("%T", v)
}
