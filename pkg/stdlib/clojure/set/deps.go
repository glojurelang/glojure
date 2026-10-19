package set

// map-invert uses reduce-kv, whose protocol fallback must also be available
// when an application links clojure.set into a compact AOT runtime.
import _ "github.com/glojurelang/glojure/pkg/stdlib/clojure/core/protocols"
