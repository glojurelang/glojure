package value

import "reflect"

// IsTruthy returns true if the value is truthy.
func IsTruthy(v interface{}) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			return false
		}
		return true
	}
}
