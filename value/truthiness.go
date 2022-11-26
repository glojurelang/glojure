package value

// IsTruthy returns true if the value is truthy.
func IsTruthy(v Value) bool {
	switch v := v.(type) {
	case *Bool:
		return v.Value
	case *Nil:
		return false
	default:
		return true
	}
}
