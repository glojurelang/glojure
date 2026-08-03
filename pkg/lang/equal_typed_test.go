package lang

import "testing"

type typedEqualValue struct{ value int }

func (v *typedEqualValue) Equals(other *typedEqualValue) bool {
	return other != nil && v.value == other.value
}

func TestEqualsTypedJavaShape(t *testing.T) {
	if !Equals(&typedEqualValue{value: 7}, &typedEqualValue{value: 7}) {
		t.Fatal("typed Equals method was not honored")
	}
}
