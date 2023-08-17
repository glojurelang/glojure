package genpkg

import (
	"fmt"
	"go/constant"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type (
	Options struct {
		writer io.Writer
	}

	Option func(*Options)

	export struct {
		name string
		obj  types.Object
	}
)

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}

func GenPkgs(packages []string, options ...Option) {
	opts := &Options{
		writer: os.Stdout,
	}
	for _, opt := range options {
		opt(opts)
	}

	sortedPackageNames, packageExports := collectExportedObjects(packages)
	printGeneratedCode(opts.writer, sortedPackageNames, packageExports)
}

func collectExportedObjects(packages []string) ([]string, map[string][]export) {
	var packageNames []string
	packageExports := map[string][]export{}

	fileSet := token.NewFileSet()
	sourceImporter := importer.ForCompiler(fileSet, "source", nil)

	for _, packageName := range packages {
		packageData, err := sourceImporter.Import(packageName)
		if err != nil {
			panic(err)
		}

		var exportedObjects []export
		for _, exportedObjectName := range packageData.Scope().Names() {
			object := packageData.Scope().Lookup(exportedObjectName)
			if isEligibleForExport(object, packageName) {
				exportedObjects = append(exportedObjects, export{
					name: exportedObjectName,
					obj:  object,
				})
			}
		}

		if len(exportedObjects) > 0 {
			packageExports[packageName] = exportedObjects
			packageNames = append(packageNames, packageName)
		}
	}

	sort.Strings(packageNames)

	return packageNames, packageExports
}

func isEligibleForExport(object types.Object, packageName string) bool {
	if !object.Exported() {
		return false
	}

	if _, isIllegalType := object.(*types.Builtin); isIllegalType {
		return false
	}

	return true
}

func printGeneratedCode(w io.Writer, packageNames []string, packageExports map[string][]export) {
	builder := createHeaderBuilder(packageNames)
	createFunctionBuilder(builder, packageNames, packageExports)

	formattedCode, err := format.Source([]byte(builder.String()))
	if err != nil {
		panic(err)
	}

	if _, err := w.Write(formattedCode); err != nil {
		panic(err)
	}
	if wc, ok := w.(io.Closer); ok {
		if err := wc.Close(); err != nil {
			panic(err)
		}
	}
}

func createHeaderBuilder(packageNames []string) *strings.Builder {
	builder := &strings.Builder{}

	builder.WriteString("// GENERATED FILE. DO NOT EDIT.\n")
	builder.WriteString("package gljimports\n\n")
	builder.WriteString("import (\n")

	for _, packageName := range packageNames {
		aliasName := strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(packageName)
		builder.WriteString(fmt.Sprintf("\t%s \"%s\"\n", aliasName, packageName))
	}

	builder.WriteString(")\n\n")

	return builder
}

func createFunctionBuilder(builder *strings.Builder, packageNames []string, packageExports map[string][]export) {
	builder.WriteString("func RegisterImports(_register func(string, interface{})) {\n")

	first := true
	for _, packageName := range packageNames {
		if !first {
			builder.WriteString("\n")
		}
		first = false
		addPackageSection(builder, packageName, packageExports[packageName])
	}

	builder.WriteString("}\n")
}

func addPackageSection(builder *strings.Builder, packageName string, packageExports []export) {
	builder.WriteString(fmt.Sprintf("\t// package %s\n", packageName))
	builder.WriteString("\t////////////////////////////////////////\n")

	packageAlias := replaceSpecChars(packageName)

	for _, exportedObject := range packageExports {
		declareExportedObject(builder, exportedObject, packageName, packageAlias)
	}
}

func replaceSpecChars(value string) string {
	return strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(value)
}

func declareExportedObject(builder *strings.Builder, exportedObject export, packageName string, packageAlias string) {
	globalName := fmt.Sprintf("%s.%s", packageName, exportedObject.name)
	aliasName := fmt.Sprintf("%s.%s", packageAlias, exportedObject.name)

	decl := getDeclaration(exportedObject.obj, globalName, aliasName)

	for _, line := range strings.Split(decl, "\n") {
		builder.WriteRune('\t')
		builder.WriteString(line)
		builder.WriteRune('\n')
	}
}

func getDeclaration(object types.Object, globalName string, aliasName string) string {
	switch concreteObject := object.(type) {
	case *types.Const:
		return getConstDeclaration(concreteObject, globalName, aliasName)
	case *types.Func:
		return fmt.Sprintf("_register(%q, %s)", globalName, aliasName)
	case *types.Var:
		return getVarDeclaration(concreteObject, globalName, aliasName)
	case *types.TypeName:
		return getTypeNameDeclaration(concreteObject, globalName, aliasName)
	default:
		panic(fmt.Errorf("unknown type %T (%v)", object, object))
	}
}

func getConstDeclaration(object *types.Const, globalName string, aliasName string) string {
	value := object.Val()

	switch value.Kind() {
	case constant.Bool, constant.String, constant.Complex:
		return fmt.Sprintf("_register(%q, %s)", globalName, aliasName)
	case constant.Int:
		val := aliasName
		if intType := getTypedInt(value.ExactString()); intType != "" {
			val = fmt.Sprintf("%s(%s)", intType, val)
		}
		return fmt.Sprintf("_register(%q, %s)", globalName, val)
	case constant.Float:
		return fmt.Sprintf("_register(%q, float64(%s))", globalName, aliasName)
	default:
		panic(fmt.Errorf("unknown constant type: %s", value.Kind()))
	}
}

func getVarDeclaration(object *types.Var, globalName string, aliasName string) string {
	typeName := object.Type().String()

	if typeName == "sync.Mutex" || typeName == "sync.RWMutex" {
		return fmt.Sprintf("_register(%q, &%s)", globalName, aliasName)
	}

	return fmt.Sprintf("_register(%q, %s)", globalName, aliasName)
}

func getTypeNameDeclaration(object *types.TypeName, globalName string, aliasName string) string {
	isGeneric := strings.HasSuffix(object.Type().String(), "]")

	if isGeneric {
		return ""
	}

	return fmt.Sprintf("_register(%q, reflect.TypeOf((*%s)(nil)).Elem())", globalName, aliasName)
}

// getTypedInt derives the type of the integer literal from its string
// representation. Use the smallest type >= int that can represent the value.
func getTypedInt(value string) string {
	if v, err := strconv.ParseInt(value, 10, 0); err == nil && v >= math.MinInt && v <= math.MaxInt {
		return ""
	} else if v, err := strconv.ParseInt(value, 10, 32); err == nil && v >= -2147483648 && v <= 2147483647 {
		return "int32"
	} else if v, err := strconv.ParseUint(value, 10, 32); err == nil && v <= 4294967295 {
		return "uint32"
	} else if v, err := strconv.ParseInt(value, 10, 64); err == nil && v >= -9223372036854775808 && v <= 9223372036854775807 {
		return "int64"
	} else if _, err := strconv.ParseUint(value, 10, 64); err == nil {
		return "uint64"
	} else {
		panic(fmt.Errorf("cannot determine type of integer literal %s", value))
	}
}
