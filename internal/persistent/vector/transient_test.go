package vector

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestTransient(t *testing.T) {
	trans := NewTransient(&vector{})

	const n = 10000

	for i := 0; i < n; i++ {
		trans.Conj(i)
	}
	vec := trans.Persistent()
	for i := 0; i < n; i++ {
		val, ok := vec.Index(i)
		if !ok {
			t.Errorf("Index %d not found", i)
		}
		if val != i {
			t.Errorf("Index %d has value %d, expected %d", i, val, i)
		}
	}
}

func TestTransientNew(t *testing.T) {
	vec := New()

	vec = vec.Conj(1)
	if val, ok := vec.Index(0); !ok || val != 1 {
		t.Errorf("Index 0 has value %v, expected 1", val)
	}
}

func FuzzTransient(f *testing.F) {
	f.Add([]byte(`[
		["conj", 1],
    ["pop"],
    ["conj", 2],
    ["persistent"]
  ]`))
	f.Add([]byte(`[
		["conj", 1],
    ["pop"],
    ["conj", 2],
    ["persistent"],
    ["assoc", 0, 0],
    ["conj", 3]
  ]`))
	f.Add([]byte(`[` +
		strings.Repeat(`["conj", 2],`, 100) +
		`
    ["pop"],
    ["conj", 2],
    ["assoc", 0, 0],
    ["persistent"],
    ["conj", 3],
    ["pop"]
  ]`))

	f.Fuzz(func(t *testing.T, buf []byte) {
		var jsOps [][]any
		if err := json.Unmarshal(buf, &jsOps); err != nil {
			t.Skip()
		}

		ops := parseOps(t, jsOps)

		persistent := false

		trans := NewTransient(New())
		// oracle is a reference implementation
		var oracle []any
		for _, op := range ops {
			if err := op.apply(t, trans, &oracle, &persistent); err != nil {
				t.Fatalf("error applying op %v: %v\n%s", op, err, string(buf))
			}

			if len(oracle) != trans.Count() {
				t.Errorf("expected count %d, got %d", len(oracle), trans.Count())
			}

			for i := 0; i < len(oracle); i++ {
				val, ok := trans.Index(i)
				if !ok {
					t.Errorf("index %d not found", i)
				}
				if val != oracle[i] {
					t.Errorf("index %d has value %v, expected %v", i, val, oracle[i])
				}
			}
		}
	})
}

type testOp struct {
	name string
	arg  any
	arg2 any
}

func (op testOp) apply(t *testing.T, vec *Transient, oracle *[]any, persistent *bool) (err error) {
	defer func() {
		if *persistent && op.name != "persistent" {
			rerr := recover()
			if rerr == nil {
				err = fmt.Errorf("expected panic")
			}
		}
	}()
	switch op.name {
	case "conj":
		vec.Conj(op.arg)
		*oracle = append(*oracle, op.arg)
	case "pop":
		vec.Pop()
		if len(*oracle) > 0 {
			*oracle = (*oracle)[:len(*oracle)-1]
		}
	case "assoc":
		i := int(op.arg.(float64))
		vec.Assoc(i, op.arg2)
		if i == len(*oracle) {
			*oracle = append(*oracle, op.arg2)
		} else if i >= 0 && i < len(*oracle) {
			(*oracle)[i] = op.arg2
		}
	case "persistent":
		*persistent = true
		v := vec.Persistent()
		if len(*oracle) != v.Len() {
			t.Errorf("expected persistent count %d, got %d", len(*oracle), v.Len())
		}
		for i := 0; i < v.Len(); i++ {
			val, ok := v.Index(i)
			if !ok {
				return fmt.Errorf("index %d not found in persistent", i)
			}
			if val != (*oracle)[i] {
				return fmt.Errorf("persistent index %d has value %v, expected %v", i, val, (*oracle)[i])
			}
		}
	default:
		panic(fmt.Errorf("unknown op %q", op.name))
	}
	return nil
}

func parseOps(t *testing.T, jsOps [][]any) []testOp {
	var ops []testOp
	for _, jsOp := range jsOps {
		if len(jsOp) == 0 {
			continue
		}
		var op testOp
		switch jsOp[0] {
		case "conj":
			if len(jsOp) != 2 {
				t.Skip()
			}
			op.name = "conj"
			f, ok := jsOp[1].(float64)
			if !ok {
				continue
			}
			op.arg = f
		case "pop":
			if len(jsOp) != 1 {
				continue
			}
			op.name = "pop"
		case "assoc":
			if len(jsOp) != 3 {
				continue
			}
			op.name = "assoc"
			f1, ok := jsOp[1].(float64)
			if !ok {
				continue
			}
			op.arg = f1
			f2, ok := jsOp[2].(float64)
			if !ok {
				continue
			}
			op.arg2 = f2
		case "persistent":
			if len(jsOp) != 1 {
				continue
			}
			op.name = "persistent"
		default:
			continue
		}

		ops = append(ops, op)
	}
	return ops
}
