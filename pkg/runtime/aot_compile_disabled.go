//go:build glj_aot_runtime

package runtime

import (
	"fmt"
	"io/fs"
)

func compileNSToFile(_ fs.FS, scriptBase string) {
	panic(fmt.Errorf(
		"cannot compile %s: Go source generation is unavailable in a glj_aot_runtime build",
		scriptBase,
	))
}
