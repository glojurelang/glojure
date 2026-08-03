// Package date exposes the small java.util.Date surface needed by hosted
// Clojure code. Values store milliseconds from the Unix epoch.
package date

import (
	"fmt"
	"reflect"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type Date struct {
	millis int64
}

func New(args ...any) any {
	switch len(args) {
	case 0:
		return &Date{millis: time.Now().UnixMilli()}
	case 1:
		return &Date{millis: toInt64(args[0])}
	default:
		panic(fmt.Sprintf("Date/new: wrong number of args (%d)", len(args)))
	}
}

func (d *Date) GetTime() int64 { return d.millis }

func (d *Date) Compare(other any) int {
	o, ok := other.(*Date)
	if !ok {
		panic(fmt.Sprintf("Date/compareTo: expected Date, got %T", other))
	}
	switch {
	case d.millis < o.millis:
		return -1
	case d.millis > o.millis:
		return 1
	default:
		return 0
	}
}

func (d *Date) CompareTo(other *Date) int32 { return int32(d.Compare(other)) }
func (d *Date) Equals(other any) bool {
	value, ok := other.(*Date)
	return ok && d.millis == value.millis
}

// ParseInstantDate implements Clojure's built-in #inst reader using Go's
// RFC3339 parser and returns the java.util.Date compatibility value.
func ParseInstantDate(value any) any {
	text, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("#inst data reader expected string, got %T", value))
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		panic(err)
	}
	return &Date{millis: parsed.UnixMilli()}
}

func init() {
	class := lang.NewClass(reflect.TypeOf((*Date)(nil)), "java.util.Date")
	pkgmap.SetHostClassPackage("Date", "java.util")
	pkgmap.SetHostClass("Date", class)
	lang.RegisterHostConstructor("java.util.Date",
		lang.FnFunc(func(args ...any) any { return New(args...) }))
	pkgmap.Set("github.com/glojurelang/glojure/pkg/javacompat/date.ParseInstantDate",
		lang.FnFunc1(ParseInstantDate))
}

func toInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		panic(fmt.Sprintf("Date/new: cannot coerce %T to long", value))
	}
}
