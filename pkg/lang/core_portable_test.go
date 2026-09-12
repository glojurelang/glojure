package lang

import (
	"reflect"
	"testing"
	"time"
)

func TestPortableMakeArray(t *testing.T) {
	array := MakeArray(reflect.TypeOf(int64(0)), int64(2), int64(3)).([][]int64)
	if len(array) != 2 || len(array[0]) != 3 {
		t.Fatalf("wrong array shape: %v", array)
	}
	array[0][1] = 42
	if array[1][1] != 0 {
		t.Fatal("array rows share storage")
	}
	if InstMillis(time.UnixMilli(-1234)) != -1234 {
		t.Fatal("instant lost milliseconds before the epoch")
	}
}

func TestPortableIteration(t *testing.T) {
	calls := 0
	it := NewIteration(
		FnFunc1(func(key any) any { calls++; return key.(int64) + 1 }),
		FnFunc1(func(value any) any { return value.(int64) <= 3 }),
		FnFunc1(func(value any) any { return value }),
		FnFunc1(func(value any) any { return value }), int64(0))
	if calls != 0 {
		t.Fatal("iteration eagerly called step")
	}
	seq := it.Seq()
	if seq.First() != int64(1) || calls != 1 {
		t.Fatal("iteration did not start lazily")
	}
	var values []any
	for ; seq != nil; seq = seq.Next() {
		values = append(values, seq.First())
	}
	if !reflect.DeepEqual(values, []any{int64(1), int64(2), int64(3)}) {
		t.Fatalf("wrong sequence: %v", values)
	}
	calls = 0
	result := it.ReduceInit(FnFunc2(func(acc, item any) any {
		return NewReduced(item)
	}), nil)
	if result != int64(1) || calls != 1 {
		t.Fatal("reduced iteration did not stop immediately")
	}
}
