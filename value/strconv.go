package value

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type printOptions struct {
	printReadably bool
}

// PrintOption is a function that configures a print operation.
type PrintOption func(*printOptions)

// PrintReadably returns a PrintOption that configures the print
// operation to print in a human-readable format.
func PrintReadably() PrintOption {
	return func(o *printOptions) {
		o.printReadably = true
	}
}

// ToString converts a value to a string. By default, any value is
// printed in a format that can be read back in by the reader. If
// printReadably is true, the output is more human-readable.
func ToString(v interface{}, opts ...PrintOption) string {
	// TODO: this function should take an io.Writer and write directly
	// to it.

	options := printOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	////////////////////////////////////////////////////////////////////////////////
	// Glojure types and special cases for native, basic types.
	switch v := v.(type) {
	case nil:
		return "nil"
	case string:
		if options.printReadably {
			return v
		}
		// NB: java does not support \x escape sequences, but go does.  this
		// results in a difference in the output of the string from Clojure
		// if such characters make it into the string. We will escape them
		// but Clojure on the JVM will not.
		return strconv.Quote(v)
	case Char:
		if options.printReadably {
			return string(v)
		}
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d.0", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case uint64, uint32, uint16, uint8, uint, int64, int32, int16, int8, int:
		return fmt.Sprintf("%d", v)
	}

	////////////////////////////////////////////////////////////////////////////////
	// if v is a Stringer, use its String method
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	////////////////////////////////////////////////////////////////////////////////
	// If v is a slice, print it as a vector
	if reflect.TypeOf(v).Kind() == reflect.Slice {
		vv := reflect.ValueOf(v)
		builder := strings.Builder{}
		builder.WriteString("[")
		for i := 0; i < vv.Len(); i++ {
			if i > 0 {
				builder.WriteString(" ")
			}
			// There is a danger here that we will recurse infinitely if the
			// slice contains itself. We should probably check for that, but
			// clojure does not.
			builder.WriteString(ToString(vv.Index(i).Interface(), opts...))
		}
		builder.WriteString("]")
		return builder.String()
	}

	if seq, ok := v.(ISeq); ok {
		builder := strings.Builder{}
		builder.WriteString("(")
		for ; !seq.IsEmpty(); seq = seq.Rest() {
			cur := seq.First()
			if builder.Len() > 1 {
				builder.WriteString(" ")
			}
			builder.WriteString(ToString(cur, opts...))
		}
		builder.WriteString(")")
		return builder.String()
	}

	return fmt.Sprintf("%T", v)
}
