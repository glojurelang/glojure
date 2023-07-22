package glj

import "github.com/glojurelang/glojure/pkg/runtime"

// Read reads one object from the string s. Reads data in the edn
// format. Read panics if the string is not a valid edn object.
func Read(s string) interface{} {
	return runtime.RTReadString(s)
}
