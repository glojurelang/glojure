package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"html/template"
	"io/ioutil"
	"strings"
)

// Generate go file with methods approximating an "abstract class",
// i.e. a common implementation of a set of methods that can be added to
// a struct.

var (
	class      = flag.String("class", "", "name of class to generate. One of APersistentMap, ASeq")
	structName = flag.String("struct", "", "name of struct to add methods to")
	receiver   = flag.String("receiver", "", "name of receiver variable")
)

func main() {
	flag.Parse()

	var contents string
	switch *class {
	case "APersistentMap":
		contents = genAPersistentMap()
	case "ASeq":
		contents = genTemplate(aseqTmpl)
	default:
		panic("unknown class " + *class)
	}
	writeFile(contents)
}

// writeFile gofmts contents, then writes it to a file in the current
// directory named after the struct: "{lowercase struct name}_{lowercase class}.go"
func writeFile(contents string) {
	formatted, err := format.Source([]byte(contents))
	if err != nil {
		panic(err)
	}
	filename := fmt.Sprintf("%s_%s.go", strings.ToLower(*structName), strings.ToLower(*class))
	if err := ioutil.WriteFile(filename, formatted, 0644); err != nil {
		panic(err)
	}
}

type Data struct {
	Receiver string
	Struct   string
}

func genAPersistentMap() string {
	w := bytes.NewBuffer(nil)
	err := template.Must(template.New("APersistentMap").Parse(
		`// GENERATED CODE. DO NOT EDIT
package value

import (
	"errors"
	"fmt"
)

func ({{.Receiver}} *{{.Struct}}) Conj(x interface{}) Conjer {
	switch x := x.(type) {
	case *MapEntry:
		return {{.Receiver}}.Assoc(x.Key(), x.Val()).(Conjer)
	case IPersistentVector:
		if {{.Receiver}}.Count() != 2 {
			panic("vector arg to map conj must be a pair")
		}
		return m.Assoc(MustNth(x, 0), MustNth(x, 1)).(Conjer)
	}

	var ret Conjer = m
	for seq := Seq(x); seq != nil; seq = seq.Next() {
		ret = ret.Conj(seq.First().(*MapEntry))
	}
	return ret
}

func ({{.Receiver}} *{{.Struct}}) ContainsKey(key interface{}) bool {
	return {{.Receiver}}.EntryAt(key) != nil
}

func ({{.Receiver}} *{{.Struct}}) AssocEx(k, v interface{}) IPersistentMap {
	if {{.Receiver}}.ContainsKey(k) {
		panic(errors.New("key already present"))
	}
	return {{.Receiver}}.Assoc(k, v).(IPersistentMap)
}

func ({{.Receiver}} *{{.Struct}}) Equal(v2 interface{}) bool {
	return mapEquals({{.Receiver}}, v2)
}

func ({{.Receiver}} *{{.Struct}}) IsEmpty() bool {
	return {{.Receiver}}.Count() == 0
}

func ({{.Receiver}} *{{.Struct}}) ValAt(key interface{}) interface{} {
	return {{.Receiver}}.ValAtDefault(key, nil)
}

// IFn methods

func ({{.Receiver}} *{{.Struct}}) Invoke(args ...interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("map apply expects 1 argument, got %d", len(args)))
	}

	return {{.Receiver}}.ValAt(args[0])
}

func ({{.Receiver}} *{{.Struct}}) ApplyTo(args ISeq) interface{} {
	return {{.Receiver}}.Invoke(seqToSlice(args)...)
}


`)).Execute(w, &Data{
		Receiver: *receiver,
		Struct:   *structName,
	})
	if err != nil {
		panic(err)
	}
	return w.String()
}

func genTemplate(tmpl string) string {
	w := bytes.NewBuffer(nil)
	err := template.Must(template.New(*structName).Parse(tmpl)).Execute(w, &Data{
		Receiver: *receiver,
		Struct:   *structName,
	})
	if err != nil {
		panic(err)
	}
	return w.String()
}

const (
	aseqTmpl = `// GENERATED CODE. DO NOT EDIT
package value

func ({{.Receiver}} *{{.Struct}}) xxx_sequential() {}

func ({{.Receiver}} *{{.Struct}}) More() ISeq {
	sq := {{.Receiver}}.Next()
	if sq == nil {
		return emptyList
	}
	return sq
}

func ({{.Receiver}} *{{.Struct}}) Seq() ISeq {
	return {{.Receiver}}
}

func ({{.Receiver}} *{{.Struct}}) Meta() IPersistentMap {
	return {{.Receiver}}.meta
}

func ({{.Receiver}} *{{.Struct}}) WithMeta(meta IPersistentMap) interface{} {
	if Equal({{.Receiver}}.meta, meta) {
		return {{.Receiver}}
	}
	cpy := *{{.Receiver}}
	cpy.meta = meta
	return &cpy
}
`
)
