package util

import (
	"github.com/glojurelang/glojure/value"
)

func Seq(x interface{}) value.ISeq {
	// TODO: deduplicate with value/seq.go
	return value.Seq(x)
}
