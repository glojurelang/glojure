package lang

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ToString converts a value to a string a la Java's .toString method.
func ToString(v interface{}) string {
	switch v := v.(type) {
	case nil:
		return "nil"
	case string:
		return v
	case Char:
		return string(v)
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
	case *BigInt:
		return v.String()
	case *BigDecimal:
		return v.String()
	}

	////////////////////////////////////////////////////////////////////////////////
	// if v is a Stringer, use its String method
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	////////////////////////////////////////////////////////////////////////////////
	// If v is a slice, print it as a vector
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		builder := strings.Builder{}
		builder.WriteString("[")
		for i := 0; i < vv.Len(); i++ {
			if i > 0 {
				builder.WriteString(" ")
			}
			// There is a danger here that we will recurse infinitely if the
			// slice contains itself. We should probably check for that, but
			// clojure does not.
			builder.WriteString(ToString(vv.Index(i).Interface()))
		}
		builder.WriteString("]")
		return builder.String()
	}

	// if seq, ok := v.(ISeq); ok {
	// 	builder := strings.Builder{}
	// 	builder.WriteString("(")
	// 	for ; seq != nil; seq = seq.Next() {
	// 		cur := seq.First()
	// 		if builder.Len() > 1 {
	// 			builder.WriteString(" ")
	// 		}
	// 		builder.WriteString(ToString(cur))
	// 	}
	// 	builder.WriteString(")")
	// 	return builder.String()
	// }

	return fmt.Sprintf("#object[%T]", v)
}

// RTPrintString corresponds to Clojure's RT.printString.
func PrintString(v interface{}) string {
	sb := strings.Builder{}
	Print(v, &sb)
	return sb.String()
}

// Print prints a value to the given io.Writer. Corresponds to
// Clojure's RT.print.
func Print(x interface{}, w io.Writer) {
	if VarPrintInitialized.IsBound() && BooleanCast(VarPrintInitialized.Deref()) {
		VarPrOn.Invoke(x, w)
		return
	}
	readably := BooleanCast(VarPrintReadably.Deref())

	if IsNil(x) {
		io.WriteString(w, "nil")
	} else if seq, ok := x.(ISeq); ok {
		io.WriteString(w, "(")
		for ; seq != nil; seq = seq.Next() {
			Print(seq.First(), w)
			if seq.Next() != nil {
				io.WriteString(w, " ")
			}
		}
		io.WriteString(w, ")")
	} else if s, ok := x.(string); ok {
		if !readably {
			io.WriteString(w, s)
		} else {
			io.WriteString(w, strconv.Quote(s))
		}
	} else if m, ok := x.(IPersistentMap); ok {
		io.WriteString(w, "{")
		for seq := m.Seq(); seq != nil; seq = seq.Next() {
			e := seq.First().(IMapEntry)
			Print(e.Key(), w)
			io.WriteString(w, " ")
			Print(e.Val(), w)
			if seq.Next() != nil {
				io.WriteString(w, ", ")
			}
		}
		io.WriteString(w, "}")
	} else if v, ok := x.(IPersistentVector); ok {
		io.WriteString(w, "[")
		for i := 0; i < v.Count(); i++ {
			Print(MustNth(v, i), w)
			if i < v.Count()-1 {
				io.WriteString(w, " ")
			}
		}
		io.WriteString(w, "]")
	} else if s, ok := x.(IPersistentSet); ok {
		io.WriteString(w, "#{")
		for seq := s.Seq(); seq != nil; seq = seq.Next() {
			Print(seq.First(), w)
			if seq.Next() != nil {
				io.WriteString(w, " ")
			}
		}
		io.WriteString(w, "}")
	} else if c, ok := x.(Char); ok {
		if !readably {
			io.WriteString(w, string(c))
		} else {
			io.WriteString(w, CharLiteralFromRune(rune(c)))
		}
	} else if v, ok := x.(*BigDecimal); ok && readably {
		io.WriteString(w, v.String())
		io.WriteString(w, "M")
	} else if v, ok := x.(*BigInt); ok && readably {
		io.WriteString(w, v.String())
		io.WriteString(w, "N")
	} else if v, ok := x.(*Var); ok {
		io.WriteString(w, "#=(var "+v.Namespace().Name().Name()+"/"+v.Symbol().Name()+")")
	} else if v, ok := x.(*regexp.Regexp); ok {
		io.WriteString(w, "#\""+v.String()+"\"")
	} else {
		io.WriteString(w, ToString(x))
	}
}
