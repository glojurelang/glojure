package main

import (
	"flag"
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"strings"
)

var (
	defaultPackages = []string{
		"bytes",
		"context",
		"errors",
		"flag",
		"fmt",
		"io",
		"io/fs",
		"io/ioutil",
		"math",
		"math/big",
		"math/rand",
		"net/http",
		"os",
		"os/exec",
		"os/signal",
		"regexp",
		"reflect",
		"runtime",
		"sort",
		"strconv",
		"strings",
		"sync",
		"sync/atomic",
		"time",
		"unicode",

		"github.com/glojurelang/glojure/pkg/runtime",
		"github.com/glojurelang/glojure/pkg/lang",
	}
)

var (
	packages = flag.String("packages", "", "comma separated list of packages to import")
)

func main() {
	flag.Parse()

	pkgs := defaultPackages
	if *packages != "" {
		pkgs = strings.Split(*packages, ",")
	}

	builder := &strings.Builder{}
	builder.WriteString("// GENERATED FILE. DO NOT EDIT.\n")
	builder.WriteString("package gljimports\n\n")
	builder.WriteString("import (\n")
	importedReflect := false
	for _, pkg := range pkgs {
		if pkg == "reflect" {
			importedReflect = true
		}
		aliasName := strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(pkg)
		builder.WriteString(fmt.Sprintf("\t%s \"%s\"\n", aliasName, pkg))
	}
	// import reflect
	if !importedReflect {
		builder.WriteString("\t\"reflect\"\n")
	}
	builder.WriteString(")\n\n")

	builder.WriteString("func RegisterImports(_register func(string, interface{})) {\n")
	for i, pkgName := range pkgs {
		if i > 0 {
			builder.WriteRune('\n')
		}
		builder.WriteString(fmt.Sprintf("\t// package %s\n", pkgName))
		builder.WriteString(fmt.Sprintf("\t%s\n", strings.Repeat("//", 20)))

		pkg, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(pkgName)
		if err != nil {
			panic(err)
		}

		for _, declName := range pkg.Scope().Names() {
			obj := pkg.Scope().Lookup(declName)
			if !obj.Exported() {
				continue
			}

			glojureDeclName := fmt.Sprintf("%s.%s", pkgName, declName)

			pkgImportName := strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(pkgName)
			qualifiedName := fmt.Sprintf("%s.%s", pkgImportName, declName)

			var decl string

			switch obj.(type) {
			case *types.Func, *types.Const, *types.Var:
				// handle some special cases in a hacky way for now.
				switch qualifiedName {
				case "math.MaxUint":
					decl = fmt.Sprintf("_register(%q, uint(%s))", glojureDeclName, qualifiedName)
				case "math.MaxUint64":
					decl = fmt.Sprintf("_register(%q, uint64(%s))", glojureDeclName, qualifiedName)
				default:
					decl = fmt.Sprintf("_register(%q, %s)", glojureDeclName, qualifiedName)
				}
			case *types.TypeName:
				// continue if this is a generic type
				if strings.HasSuffix(obj.Type().String(), "]") {
					continue
				}

				decl = fmt.Sprintf("_register(%q, reflect.TypeOf((*%s)(nil)).Elem())", glojureDeclName, qualifiedName)
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
