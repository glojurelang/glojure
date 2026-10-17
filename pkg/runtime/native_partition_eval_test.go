//go:build !glj_aot_runtime

package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativePartitionPreservesOtherAritiesAndSizes(t *testing.T) {
	for _, test := range []struct{ code, want string }{
		{`(mapv vec (partition 2 [0 1 2 3 4]))`, `[[0 1] [2 3]]`},
		{`(mapv vec (partition 2 1 [0 1 2]))`, `[[0 1] [1 2]]`},
		{`(mapv vec (partition 3 3 [:pad] [0 1]))`, `[[0 1 :pad]]`},
		{`(mapv vec (take 2 (partition 0 [1])))`, `[[] []]`},
		{`(vec (partition -1 [1 2]))`, `[]`},
		{`(vec (partition 2.5 [0 1 2 3]))`, `[]`},
	} {
		if got := lang.PrintString(ReadEval(test.code)); got != test.want {
			t.Fatalf("%s = %s, want %s", test.code, got, test.want)
		}
	}
}
