// Package boolean exposes JVM-faithful java.lang.Boolean equivalents for code
// running on glojure.
package boolean

import (
	"fmt"

	jbool "github.com/gloathub/gojava/boolean"
	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/pkgmap"
)

const pkg = "github.com/gloathub/glojure/pkg/javacompat/boolean"

const (
	TRUE  = jbool.TRUE
	FALSE = jbool.FALSE
)

func ParseBoolean(x any) bool { return jbool.ParseBoolean(toString(x)) }

// ValueOf accepts a string ("true" ci -> true; anything else -> false) or
// any value coerced to bool. Mirrors Boolean.valueOf(String|boolean).
func ValueOf(x any) bool {
	switch v := x.(type) {
	case string:
		return jbool.ValueOfString(v)
	default:
		return toBool(v)
	}
}

func ToString(x any) string { return jbool.ToString(toBool(x)) }

func Compare(x, y any) int32 { return jbool.Compare(toBool(x), toBool(y)) }

func LogicalAnd(x, y any) bool { return jbool.LogicalAnd(toBool(x), toBool(y)) }
func LogicalOr(x, y any) bool  { return jbool.LogicalOr(toBool(x), toBool(y)) }
func LogicalXor(x, y any) bool { return jbool.LogicalXor(toBool(x), toBool(y)) }

func GetBoolean(x any) bool { return jbool.GetBoolean(toString(x)) }

func register(jvmName, goName string, v any) {
	pkgmap.Set(pkg+"."+goName, v)
	pkgmap.Set("Boolean."+jvmName, v)
}

func init() {
	register("TRUE", "TRUE", TRUE)
	register("FALSE", "FALSE", FALSE)

	register("parseBoolean", "ParseBoolean", lang.FnFunc(func(args ...any) any { return ParseBoolean(args[0]) }))
	register("valueOf", "ValueOf", lang.FnFunc(func(args ...any) any { return ValueOf(args[0]) }))
	register("toString", "ToString", lang.FnFunc(func(args ...any) any { return ToString(args[0]) }))

	register("compare", "Compare", lang.FnFunc(func(args ...any) any { return Compare(args[0], args[1]) }))

	register("logicalAnd", "LogicalAnd", lang.FnFunc(func(args ...any) any { return LogicalAnd(args[0], args[1]) }))
	register("logicalOr", "LogicalOr", lang.FnFunc(func(args ...any) any { return LogicalOr(args[0], args[1]) }))
	register("logicalXor", "LogicalXor", lang.FnFunc(func(args ...any) any { return LogicalXor(args[0], args[1]) }))

	register("getBoolean", "GetBoolean", lang.FnFunc(func(args ...any) any { return GetBoolean(args[0]) }))
}

func toString(x any) string {
	switch v := x.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func toBool(x any) bool {
	switch v := x.(type) {
	case bool:
		return v
	case nil:
		return false
	}
	panic(fmt.Sprintf("cannot coerce %T to bool", x))
}
