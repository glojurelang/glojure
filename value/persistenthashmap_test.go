package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPersistentHashMap(t *testing.T) {
	var m Associative = NewPersistentHashMap()
	m = m.Assoc(nil, 1)
	assert.Equal(t, 1, m.ValAt(nil))

	m = NewPersistentHashMap()
	for i := 0; i < 1000; i++ {
		m = m.Assoc(i, i)
	}
	for i := 0; i < 1000; i++ {
		assert.Equal(t, i, m.ValAt(i))
	}
}
