//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestCompiledNumbersAcceptHostIntegers(t *testing.T) {
	for _, operator := range []struct {
		name   string
		native func(any, any) any
	}{
		{"+", lang.Numbers.Add},
		{"-", lang.Numbers.Minus},
		{"*", lang.Numbers.Multiply},
		{"quot", lang.Numbers.Quotient},
		{"<", func(a, b any) any { return lang.Numbers.Lt(a, b) }},
		{">", func(a, b any) any { return lang.Numbers.Gt(a, b) }},
	} {
		fn := ReadEval(fmt.Sprintf("(fn [x y] (%s x y))", operator.name)).(*Fn)
		for _, args := range [][2]any{
			{int(7), int(3)}, {int64(7), int(3)},
			{int(7), int64(3)}, {float64(7), int64(3)},
		} {
			want := operator.native(args[0], args[1])
			got := fn.Invoke2(args[0], args[1])
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s(%T,%T) = %v (%T), want %v (%T)",
					operator.name, args[0], args[1], got, got, want, want)
			}
		}
	}
}

func TestCompiledHostIntegerOverflow(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("Go int cannot overflow an int64 here")
	}
	fn := ReadEval(`(fn [n] (+ n 1))`).(*Fn)
	defer func() {
		if recover() == nil {
			t.Fatal("host integer overflow was not checked")
		}
	}()
	fn.Invoke1(int(^uint(0) >> 1))
}
