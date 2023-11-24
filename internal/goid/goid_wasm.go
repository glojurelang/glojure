//go:build wasm

package goid

import (
	"bytes"
	"runtime"
	"strconv"
)

var (
	goroutinePrefix = []byte("goroutine ")
)

func Get() int64 {
	buf := make([]byte, 32)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	// goroutine 1 [running]: ...

	if !bytes.HasPrefix(buf, goroutinePrefix) {
		panic("unexpected goroutine stack format, missing prefix")
	}
	buf = buf[len(goroutinePrefix):]

	i := bytes.IndexByte(buf, ' ')
	if i < 0 {
		panic("unexpected goroutine stack format, missing space")
	}

	id, err := strconv.Atoi(string(buf[:i]))
	if err != nil {
		panic("unexpected goroutine stack format, invalid id")
	}
	return int64(id)
}
