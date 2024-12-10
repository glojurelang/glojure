package lang

import (
	"fmt"
	"strings"
)

func Compare(a, b any) int {
	if s1, ok := a.(string); ok {
		if s2, ok := b.(string); ok {
			return strings.Compare(s1, s2)
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	} else if i1, ok := a.(int); ok {
		if i2, ok := b.(int); ok {
			if i1 > i2 {
				return 1
			}
			if i1 < i2 {
				return -1
			}
			return 0
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	} else if i1, ok := a.(int32); ok {
		if i2, ok := b.(int32); ok {
			if i1 > i2 {
				return 1
			}
			if i1 < i2 {
				return -1
			}
			return 0
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	} else if i1, ok := a.(int64); ok {
		if i2, ok := b.(int64); ok {
			if i1 > i2 {
				return 1
			}
			if i1 < i2 {
				return -1
			}
			return 0
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	} else if i1, ok := a.(float32); ok {
		if i2, ok := b.(float32); ok {
			if i1 > i2 {
				return 1
			}
			if i1 < i2 {
				return -1
			}
			return 0
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	} else if i1, ok := a.(float64); ok {
		if i2, ok := b.(float64); ok {
			if i1 > i2 {
				return 1
			}
			if i1 < i2 {
				return -1
			}
			return 0
		}
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot compare unrelated types: %T, %T", a, b)))
	}

	panic(NewIllegalArgumentError(fmt.Sprintf("unable to compare types: %T, %T", a, b)))
}
