package main

import (
	"fmt"
	"go/importer"
	"go/types"
	"strings"
)

var (
	packages = []string{
		"fmt",
		"time",
		"regexp",
		"strings", "bytes", "net/http", "io", "io/ioutil", "io/fs",
	}
)

func main() {
	builder := &strings.Builder{}
	builder.WriteString("package gljimports\n\n")
	builder.WriteString("import (\n")
	for _, pkg := range packages {
		aliasName := strings.NewReplacer(".", "_", "/", "_").Replace(pkg)
		builder.WriteString(fmt.Sprintf("\t%s \"%s\"\n", aliasName, pkg))
	}
	// import reflect
	builder.WriteString("\t\"reflect\"\n")
	// import "github.com/jfhamlin/muscrat/internal/pkg/mratlang/value"
	builder.WriteString("\t\"github.com/jfhamlin/muscrat/internal/pkg/mratlang/value\"\n")

	builder.WriteString(")\n\n")

	builder.WriteString("func RegisterImports(_register func(string, interface{})) {\n")
	for i, pkgName := range packages {
		if i > 0 {
			builder.WriteRune('\n')
		}
		builder.WriteString(fmt.Sprintf("\t// package %s\n", pkgName))
		builder.WriteString(fmt.Sprintf("\t%s\n", strings.Repeat("//", 20)))

		pkg, err := importer.Default().Import(pkgName)
		if err != nil {
			panic(err)
		}

		for _, declName := range pkg.Scope().Names() {
			obj := pkg.Scope().Lookup(declName)
			if !obj.Exported() {
				continue
			}

			glojureDeclName := fmt.Sprintf("go/%s.%s", pkgName, declName)

			pkgImportName := strings.NewReplacer(".", "_", "/", "_").Replace(pkgName)
			qualifiedName := fmt.Sprintf("%s.%s", pkgImportName, declName)

			var decl string

			switch obj.(type) {
			case *types.Func, *types.Const, *types.Var:
				decl = fmt.Sprintf("_register(%q, value.NewGoVal(%s))", glojureDeclName, qualifiedName)
			case *types.TypeName:
				decl = fmt.Sprintf(`{
	var x %s
	_register(%q, value.NewGoTyp(reflect.TypeOf(x)))
}`, qualifiedName, glojureDeclName)
			default:
				panic(fmt.Sprintf("unknown type %T", obj))
			}
			for _, line := range strings.Split(decl, "\n") {
				builder.WriteRune('\t')
				builder.WriteString(line)
				builder.WriteRune('\n')
			}
		}
	}
	builder.WriteString("}\n")
	fmt.Print(builder.String())
}
