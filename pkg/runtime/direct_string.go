package runtime

import (
	"strings"
)

// CoreString implements clojure.core/str's single-value conversion as a
// concrete Go string. It is used by fused string collection pipelines.
func CoreString(value any) string {
	return nativeStrValue(value)
}

// ConcatStringParts implements (apply str parts) for a compiler-owned parts
// buffer. Values are deliberately converted only here, matching Clojure's
// left-to-right argument traversal and the timing of user ToString calls.
func ConcatStringParts(parts []any) string {
	var builder strings.Builder
	for _, value := range parts {
		builder.WriteString(nativeStrValue(value))
	}
	return builder.String()
}
