package vector

import "testing"

func TestTransient(t *testing.T) {
	trans := newTransient(&vector{})

	const n = 10000

	for i := 0; i < n; i++ {
		trans.conj(i)
	}
	vec := trans.persistent()
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
