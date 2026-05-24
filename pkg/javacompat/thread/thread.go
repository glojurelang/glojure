// Package thread exposes JVM-faithful java.lang.Thread equivalents for code
// running on glojure. Only the static methods whose semantics map cleanly
// onto Go's runtime are registered here; instance methods (start, join,
// interrupt, ...) would require a synthetic per-goroutine identity layer
// and are out of scope for now.
package thread

import (
	"fmt"

	jthread "github.com/gloathub/gojava/thread"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/thread"

// Sleep mirrors java.lang.Thread.sleep. Accepts (millis) or (millis, nanos);
// both arguments are coerced from any int-like value glojure may pass.
func Sleep(args ...any) any {
	switch len(args) {
	case 1:
		jthread.Sleep(toInt64(args[0]))
	case 2:
		jthread.Sleep(toInt64(args[0]), toInt64(args[1]))
	default:
		panic("Thread/sleep takes 1 or 2 arguments")
	}
	return nil
}

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Thread."+jvmName, v)
}

func init() {
	register("sleep", "Sleep", lang.FnFunc(func(args ...any) any { return Sleep(args...) }))
}

func toInt64(x any) int64 {
	switch v := x.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	}
	panic(fmt.Sprintf("cannot coerce %T to int64", x))
}
