package glj

import (
	"strings"

	"github.com/glojurelang/glojure/reader"
)

// Read reads one object from the string s. Reads data in the edn
// format. Read panics if the string is not a valid edn object.
func Read(s string) interface{} {
	// TODO: use read-string directly once it works.
	rdr := reader.New(strings.NewReader(s), reader.WithGetCurrentNS(func() string { return "user" }))
	res, err := rdr.ReadOne()
	if err != nil {
		panic(err)
	}
	return res
}
