package value

// IsTruthy returns true if the value is truthy.
func IsTruthy(v interface{}) bool {
	switch v := v.(type) {
	case bool:
		return v
	case nil:
		return false
	default:
		return true
	}
}
