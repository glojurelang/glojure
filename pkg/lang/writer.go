package lang

import (
	"fmt"
	"io"
)

// AppendWriter is a shim for clojure.core's use of Java's append()
// method, which is not available on go's io.Writer interface.
func AppendWriter(w io.Writer, v interface{}) io.Writer {
	var err error
	switch v := v.(type) {
	case string:
		_, err = w.Write([]byte(v))
	case []byte:
		_, err = w.Write(v)
	case rune:
		_, err = w.Write([]byte{byte(v)})
	case Char:
		_, err = w.Write([]byte{byte(v)})
	default:
		err = fmt.Errorf("unsupported type %T", v)
	}

	if err != nil {
		panic(err)
	}
	return w
}

func WriteWriter(w io.Writer, v interface{}) io.Writer {
	var err error
	switch v := v.(type) {
	case string:
		_, err = w.Write([]byte(v))
	case []byte:
		_, err = w.Write(v)
	default:
		err = fmt.Errorf("unsupported type %T", v)
	}
	if err != nil {
		panic(err)
	}
	return w
}
