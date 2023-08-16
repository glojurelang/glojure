package main

import (
	"flag"
	"fmt"
	"go/constant"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

var defaultPackages = []string{
	"archive/tar",
	"archive/zip",
	"bufio",
	"bytes",
	"compress/bzip2",
	"compress/flate",
	"compress/gzip",
	"compress/lzw",
	"compress/zlib",
	"container/heap",
	"container/list",
	"container/ring",
	"context",
	"crypto",
	"crypto/aes",
	"crypto/cipher",
	"crypto/des",
	"crypto/dsa",
	"crypto/ecdsa",
	"crypto/ed25519",
	"crypto/elliptic",
	"crypto/hmac",
	"crypto/md5",
	"crypto/rand",
	"crypto/rc4",
	"crypto/rsa",
	"crypto/sha1",
	"crypto/sha256",
	"crypto/sha512",
	"crypto/subtle",
	"crypto/tls",
	"crypto/x509",
	"crypto/x509/pkix",
	"database/sql",
	"database/sql/driver",
	"debug/buildinfo",
	"debug/dwarf",
	"debug/elf",
	"debug/gosym",
	"debug/macho",
	"debug/pe",
	"debug/plan9obj",
	"embed",
	"encoding",
	"encoding/ascii85",
	"encoding/asn1",
	"encoding/base32",
	"encoding/base64",
	"encoding/binary",
	"encoding/csv",
	"encoding/gob",
	"encoding/hex",
	"encoding/json",
	"encoding/pem",
	"encoding/xml",
	"errors",
	"expvar",
	"flag",
	"fmt",
	"go/ast",
	"go/build",
	"go/build/constraint",
	"go/constant",
	"go/doc",
	"go/doc/comment",
	"go/format",
	"go/importer",
	"go/parser",
	"go/printer",
	"go/scanner",
	"go/token",
	"go/types",
	"hash",
	"hash/adler32",
	"hash/crc32",
	"hash/crc64",
	"hash/fnv",
	"hash/maphash",
	"html",
	"html/template",
	"image",
	"image/color",
	"image/color/palette",
	"image/draw",
	"image/gif",
	"image/jpeg",
	"image/png",
	"index/suffixarray",
	"io",
	"io/fs",
	"io/ioutil",
	"log",
	"math",
	"math/big",
	"math/bits",
	"math/cmplx",
	"math/rand",
	"mime",
	"mime/multipart",
	"mime/quotedprintable",
	"net",
	"net/http",
	"net/http/cgi",
	"net/http/cookiejar",
	"net/http/fcgi",
	"net/http/httptest",
	"net/http/httptrace",
	"net/http/pprof",
	"net/mail",
	"net/netip",
	"net/rpc",
	"net/rpc/jsonrpc",
	"net/smtp",
	"net/textproto",
	"net/url",
	"os",
	"os/exec",
	"os/signal",
	"os/user",
	"path",
	"path/filepath",
	"plugin",
	"reflect",
	"regexp",
	"regexp/syntax",
	"runtime",
	// "runtime/cgo", // exclude cgo because it imposes a dependency on cgo
	"runtime/debug",
	"runtime/metrics",
	"runtime/pprof",
	"runtime/race",
	"runtime/trace",
	"sort",
	"strconv",
	"strings",
	"sync",
	"sync/atomic",
	"syscall",
	"testing",
	"testing/fstest",
	"testing/iotest",
	"testing/quick",
	"text/scanner",
	"text/tabwriter",
	"text/template",
	"text/template/parse",
	"time",
	"time/tzdata",
	"unicode",
	"unicode/utf16",
	"unicode/utf8",
	"unsafe",

	"github.com/glojurelang/glojure/pkg/runtime",
	"github.com/glojurelang/glojure/pkg/lang",
}

var packagesFlag = flag.String(
	"packages",
	"",
	"comma separated list of packages to import",
)

type export struct {
	name string
	obj  types.Object
}

func main() {
	flag.Parse()

	packages := defaultPackages
	if *packagesFlag != "" {
		packages = strings.Split(*packagesFlag, ",")
	}

	sortedPackageNames, packageExports := collectExportedObjects(packages)
	printGeneratedCode(sortedPackageNames, packageExports)
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

func printGeneratedCode(packageNames []string, packageExports map[string][]export) {
	builder := createHeaderBuilder(packageNames)
	createFunctionBuilder(builder, packageNames, packageExports)

	formattedCode, err := format.Source([]byte(builder.String()))
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(formattedCode)
	os.Stdout.Sync()
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
