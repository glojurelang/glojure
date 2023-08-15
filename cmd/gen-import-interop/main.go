package main

import (
	"flag"
	"fmt"
	"go/constant"
	"go/importer"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

var (
	// we're including the entire stdlib here for now.
	defaultPackages = []string{
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
)

var (
	packages = flag.String("packages", "", "comma separated list of packages to import")
)

type (
	export struct {
		name string
		obj  types.Object
	}
)

func main() {
	flag.Parse()

	pkgs := defaultPackages
	if *packages != "" {
		pkgs = strings.Split(*packages, ",")
	}

	pkgExports := make(map[string][]export)
	pkgNames := make([]string, 0, len(pkgs))

	builder := &strings.Builder{}
	builder.WriteString("// GENERATED FILE. DO NOT EDIT.\n")
	builder.WriteString("package gljimports\n\n")
	builder.WriteString("import (\n")
	importedReflect := false
	for _, pkgName := range pkgs {
		pkg, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(pkgName)
		if err != nil {
			panic(err)
		}
		exports := make([]export, 0, len(pkgNames))
		for _, declName := range pkg.Scope().Names() {
			obj := pkg.Scope().Lookup(declName)
			if !obj.Exported() {
				continue
			}
			if _, ok := obj.(*types.Builtin); ok {
				// Builtin types can't be used as values, and so we can't map
				// them.
				continue
			}
			exports = append(exports, export{
				name: declName,
				obj:  obj,
			})
		}
		if len(exports) == 0 {
			continue
		}
		pkgExports[pkgName] = exports
		pkgNames = append(pkgNames, pkgName)

		if pkgName == "reflect" {
			importedReflect = true
		}

		aliasName := strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(pkgName)
		builder.WriteString(fmt.Sprintf("\t%s \"%s\"\n", aliasName, pkgName))
	}
	sort.Strings(pkgNames)

	// import reflect if not already imported. it's needed for
	// type.TypeOf.
	if !importedReflect {
		builder.WriteString("\t\"reflect\"\n")
	}
	builder.WriteString(")\n\n")

	builder.WriteString("func RegisterImports(_register func(string, interface{})) {\n")
	for i, pkgName := range pkgNames {
		if i > 0 {
			builder.WriteRune('\n')
		}
		builder.WriteString(fmt.Sprintf("\t// package %s\n", pkgName))
		builder.WriteString(fmt.Sprintf("\t%s\n", strings.Repeat("//", 20)))

		for _, export := range pkgExports[pkgName] {
			declName := export.name
			obj := export.obj

			glojureDeclName := fmt.Sprintf("%s.%s", pkgName, declName)

			pkgImportName := strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(pkgName)
			qualifiedName := fmt.Sprintf("%s.%s", pkgImportName, declName)

			var decl string

			switch obj := obj.(type) {
			case *types.Const:
				val := obj.Val()
				switch val.Kind() {
				case constant.Bool, constant.String, constant.Complex:
					decl = fmt.Sprintf("_register(%q, %s)", glojureDeclName, qualifiedName)
				case constant.Int:
					// TODO: derive type from constant. use the smallest type
					// possible that can hold the value.

					switch qualifiedName {
					case "math.MaxUint":
						decl = fmt.Sprintf("_register(%q, uint(%s))", glojureDeclName, qualifiedName)
					case "math.MaxUint64":
						decl = fmt.Sprintf("_register(%q, uint64(%s))", glojureDeclName, qualifiedName)
					case "math.MaxUint32":
						decl = fmt.Sprintf("_register(%q, uint32(%s))", glojureDeclName, qualifiedName)
					case "math.MaxUint16":
						decl = fmt.Sprintf("_register(%q, uint16(%s))", glojureDeclName, qualifiedName)
					case "math.MaxUint8":
						decl = fmt.Sprintf("_register(%q, uint8(%s))", glojureDeclName, qualifiedName)
					case "math.MaxInt":
						decl = fmt.Sprintf("_register(%q, int(%s))", glojureDeclName, qualifiedName)
					case "math.MaxInt64":
						decl = fmt.Sprintf("_register(%q, int64(%s))", glojureDeclName, qualifiedName)
					case "math.MaxInt32":
						decl = fmt.Sprintf("_register(%q, int32(%s))", glojureDeclName, qualifiedName)
					case "math.MaxInt16":
						decl = fmt.Sprintf("_register(%q, int16(%s))", glojureDeclName, qualifiedName)
					case "math.MaxInt8":
						decl = fmt.Sprintf("_register(%q, int8(%s))", glojureDeclName, qualifiedName)
					case "math.MinInt":
						decl = fmt.Sprintf("_register(%q, int(%s))", glojureDeclName, qualifiedName)
					case "math.MinInt64":
						decl = fmt.Sprintf("_register(%q, int64(%s))", glojureDeclName, qualifiedName)
					case "math.MinInt32":
						decl = fmt.Sprintf("_register(%q, int32(%s))", glojureDeclName, qualifiedName)
					case "math.MinInt16":
						decl = fmt.Sprintf("_register(%q, int16(%s))", glojureDeclName, qualifiedName)
					case "math.MinInt8":
						decl = fmt.Sprintf("_register(%q, int8(%s))", glojureDeclName, qualifiedName)
					case "hash_crc64.ISO":
						decl = fmt.Sprintf("_register(%q, uint64(%s))", glojureDeclName, qualifiedName)
					case "hash_crc64.ECMA":
						decl = fmt.Sprintf("_register(%q, uint64(%s))", glojureDeclName, qualifiedName)
					default:
						decl = fmt.Sprintf("_register(%q, %s)", glojureDeclName, qualifiedName)
					}
				case constant.Float:
					decl = fmt.Sprintf("_register(%q, float64(%s))", glojureDeclName, qualifiedName)
				default:
					panic(fmt.Errorf("unknown constant type: %s", val.Kind()))
				}
			case *types.Func:
				decl = fmt.Sprintf("_register(%q, %s)", glojureDeclName, qualifiedName)
			case *types.Var:
				// if obj is a sync.Mutex or sync.RWMutex, map to its address
				if obj.Type().String() == "sync.Mutex" || obj.Type().String() == "sync.RWMutex" {
					decl = fmt.Sprintf("_register(%q, &%s)", glojureDeclName, qualifiedName)
				} else {
					decl = fmt.Sprintf("_register(%q, %s)", glojureDeclName, qualifiedName)
				}
			case *types.TypeName:
				// skip generic types
				if strings.HasSuffix(obj.Type().String(), "]") {
					continue
				}

				decl = fmt.Sprintf("_register(%q, reflect.TypeOf((*%s)(nil)).Elem())", glojureDeclName, qualifiedName)
			default:
				panic(fmt.Sprintf("unknown type %T (%v)", obj, obj))
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
