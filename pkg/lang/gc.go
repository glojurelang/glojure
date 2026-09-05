package lang

import (
	"os"
	"runtime/debug"
)

// DefaultGCPercent is the garbage collector target set at startup when
// the GOGC environment variable is unset.
// Glojure programs allocate many short-lived boxed values, so the Go
// default of 100 starts a collection every few megabytes and spends a
// large share of CPU marking a heap that is mostly garbage.
// 400 trades a larger peak heap for far fewer collections.
// Setting GOGC in the environment overrides this, as it does for any
// Go program.
const DefaultGCPercent = 400

func init() {
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(DefaultGCPercent)
	}
}
