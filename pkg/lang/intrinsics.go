package lang

import "fmt"

// The functions in this file back the clojure.core intrinsics that AOT
// generated code emits in place of a call through the core Var. Each one
// mirrors the core definition exactly, so the compiler may substitute it
// whenever the Var still holds its original root.

// DerefValue is the intrinsic for (deref x). Volatiles are checked first
// because they dominate compiled code; any other IDeref follows.
func DerefValue(x any) any {
	if v, ok := x.(*Volatile); ok {
		return v.val
	}
	return derefOther(x)
}

func derefOther(x any) any {
	if ref, ok := x.(IDeref); ok {
		return ref.Deref()
	}
	panic(NewIllegalArgumentError(
		fmt.Sprintf("deref: value of type %T is not dereferenceable", x)))
}

// VReset is the intrinsic for (vreset! vol val).
func VReset(vol, val any) any {
	if v, ok := vol.(*Volatile); ok {
		v.val = val
		return val
	}
	return vol.(interface{ Reset(any) any }).Reset(val)
}

// IsVector is the intrinsic for vector?.
func IsVector(x any) bool {
	_, ok := x.(IPersistentVector)
	return ok
}

// IsString is the intrinsic for string?.
func IsString(x any) bool {
	_, ok := x.(string)
	return ok
}

// IsMap is the intrinsic for map?.
func IsMap(x any) bool {
	_, ok := x.(IPersistentMap)
	return ok
}

// IsKeyword is the intrinsic for keyword?.
func IsKeyword(x any) bool {
	_, ok := x.(Keyword)
	return ok
}
