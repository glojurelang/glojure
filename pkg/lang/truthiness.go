package lang

import "reflect"

// IsTruthy returns true if the value is truthy. The bool and nil checks are
// small enough for the Go inliner, so generated code pays a full call only
// for other types.
func IsTruthy(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if v == nil {
		return false
	}
	return isTruthyOther(v)
}

func isTruthyOther(v interface{}) bool {
	// The three types generated parsers test most often get plain type
	// assertions ahead of the switch, which the compiler turns into a
	// hash search once it has this many cases.
	if m, ok := v.(*Map); ok {
		return m != nil
	}
	if vec, ok := v.(*Vector); ok {
		return vec != nil
	}
	if _, ok := v.(int64); ok {
		return true
	}
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case string, int64, Keyword, Char, float64, int:
		return true
	case *Vector:
		return v != nil
	case *Map:
		return v != nil
	case *MetaFn:
		return v != nil
	case *Volatile:
		return v != nil
	case FnFunc1:
		return v != nil
	case FnFunc2:
		return v != nil
	case FnFunc3:
		return v != nil
	default:
		return !IsNil(v)
	}
}

// IsNil reports whether v is nil, treating a typed nil pointer as nil.
// The common runtime types are checked directly so only unusual host
// values pay for reflection.
func IsNil(v interface{}) bool {
	switch v := v.(type) {
	case nil:
		return true
	case string, int64, bool, Keyword, Char, float64, int:
		return false
	case *Vector:
		return v == nil
	case *Map:
		return v == nil
	case *MetaFn:
		return v == nil
	case *Volatile:
		return v == nil
	case *List:
		return v == nil
	case *LazySeq:
		return v == nil
	case *PersistentHashMap:
		return v == nil
	case *Record:
		return v == nil
	case IRecord:
		return false
	case *Cons:
		return v == nil
	case FnFunc1:
		return v == nil
	case FnFunc2:
		return v == nil
	case FnFunc3:
		return v == nil
	case FnFunc:
		return v == nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return true
	}
	return false
}
