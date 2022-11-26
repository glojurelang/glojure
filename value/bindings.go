package value

import "fmt"

func restFromNth(nth Nther, i int) Value {
	var result []Value
	for {
		val, ok := nth.Nth(i)
		if !ok {
			break
		}
		result = append(result, val)
		i++
	}
	return NewList(result)
}

var restSymbol = NewSymbol("&")

// Bind binds the values in val to the symbols in pattern.
func Bind(pattern Value, val Value) ([]Value, error) {
	// TODO: take a context.Context. This will allow us to cancel the
	// evaluation if it takes too long. Because a value may be an infinite
	// sequence, we need to be able to cancel the evaluation.

	var result []Value

	switch pattern := pattern.(type) {
	case *Vector:
		nther, ok := val.(Nther)
		if !ok {
			return nil, fmt.Errorf("cannot bind vector to non-nthable value")
		}
		for i := 0; i < pattern.Count(); i++ {
			// special case for &
			if pattern.ValueAt(i).Equal(restSymbol) {
				if i != pattern.Count()-2 {
					return nil, fmt.Errorf("& in binding-form must be followed by a single element")
				}
				target := pattern.ValueAt(i + 1)
				rest := restFromNth(nther, i)
				bindings, err := Bind(target, rest)
				if err != nil {
					return nil, err
				}
				result = append(result, bindings...)
				break
			}

			src, ok := nther.Nth(i)
			if !ok {
				return nil, fmt.Errorf("cannot bind vector to value with fewer elements")
			}
			bindings, err := Bind(pattern.ValueAt(i), src)
			if err != nil {
				return nil, err
			}
			result = append(result, bindings...)
		}
	case *Symbol:
		result = append(result, pattern, val)
	default:
		return nil, fmt.Errorf("cannot bind to %T", pattern)
	}

	return result, nil
}

func IsValidBinding(binding *Vector) bool {
	for i := 0; i < binding.Count(); i += 2 {
		switch binding.ValueAt(i).(type) {
		case *Symbol:
		case *Vector:
			if !IsValidBinding(binding.ValueAt(i).(*Vector)) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func MinMaxArgumentCount(binding *Vector) (int, int) {
	min := 0
	for i := 0; i < binding.Count(); i++ {
		switch b := binding.ValueAt(i).(type) {
		case *Symbol:
			if b.Equal(restSymbol) {
				return min, -1
			} else {
				min++
			}
		default:
			min++
		}
	}
	return min, min
}
