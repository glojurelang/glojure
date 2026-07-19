//go:build glj_no_aot_stdlib

package runtime

// Bootstrap builds keep source function roots intact so the standard library
// can be serialized into generated loaders.
const installNativeCoreOverrides = false
