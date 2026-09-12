package runtime

import (
	"io"
	"strings"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// ReadLine avoids buffering past the line, preserving subsequent reads from
// the same stream even when callers mix read-line with other reader APIs.
func ReadLine(input io.Reader) any {
	var line strings.Builder
	var byteBuffer [1]byte
	for {
		n, err := input.Read(byteBuffer[:])
		if n > 0 {
			if byteBuffer[0] == '\n' {
				return strings.TrimSuffix(line.String(), "\r")
			}
			line.WriteByte(byteBuffer[0])
		}
		if err == io.EOF {
			if line.Len() == 0 {
				return nil
			}
			return line.String()
		}
		if err != nil {
			panic(err)
		}
	}
}

func init() {
	pkgmap.Set("github.com/glojurelang/glojure/pkg/runtime.ReadLine", ReadLine)
}
