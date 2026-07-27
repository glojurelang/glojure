package runtime

// CoreString implements clojure.core/str's single-value conversion as a
// concrete Go string. It is used by fused string collection pipelines.
func CoreString(value any) string {
	return nativeStrValue(value)
}
