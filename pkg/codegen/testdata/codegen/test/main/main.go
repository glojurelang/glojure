package main

import (
	"fmt"

	_ "github.com/glojurelang/glojure/pkg/codegen/testdata/codegen/test"
	"github.com/glojurelang/glojure/pkg/glj"
)

func main() {
	run := glj.Var("codegen.test.loop-simple", "simple-loop")
	result := run.Invoke()
	fmt.Printf("%v (%T)\n", result, result)
}
