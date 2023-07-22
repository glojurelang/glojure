package lang

import (
	"testing"
)

func TestLazySeq(t *testing.T) {
	ls := NewLazySeq(func() interface{} { return nil })
	// should be empty
	if Seq(ls) != nil {
		t.Fail()
	}
}
