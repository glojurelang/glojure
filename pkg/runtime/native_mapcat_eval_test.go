//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeMapcatPreservesOtherArities(t *testing.T) {
	for _, tc := range []struct {
		code string
		want []any
	}{
		{`(vec (mapcat identity [[1 2] nil [] [3]]))`,
			[]any{int64(1), int64(2), int64(3)}},
		{`(vec (mapcat vector [1 2] [3 4]))`,
			[]any{int64(1), int64(3), int64(2), int64(4)}},
		{`(into [] (mapcat vector) [1 2])`,
			[]any{int64(1), int64(2)}},
	} {
		want := lang.NewVector(tc.want...)
		if got := ReadEval(tc.code); !lang.Equals(got, want) {
			t.Fatalf("%s = %v, want %v", tc.code, got, want)
		}
	}
}

func TestNativeMapcatPreservesValidation(t *testing.T) {
	for _, tc := range []struct {
		code    string
		message string
	}{
		{`(mapcat nil [1])`, "mapcat expected a function"},
		{`(mapcat identity 42)`, "mapcat expected a seqable collection"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil || !strings.Contains(fmt.Sprint(got), tc.message) {
					t.Errorf("panic = %v, want %s", got, tc.message)
				}
			}()
			ReadEval(tc.code)
		})
	}
}
