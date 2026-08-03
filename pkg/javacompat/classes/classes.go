// Package classes registers common JVM marker and runtime classes that map
// directly onto existing Glojure host types.
package classes

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func register(name, javaPackage string, typ reflect.Type) {
	pkgmap.SetHostClassPackage(name, javaPackage)
	pkgmap.SetHostClass(name, lang.NewClass(typ, javaPackage+"."+name))
}

func newBigInteger(args ...any) any {
	if len(args) < 1 || len(args) > 2 {
		panic(fmt.Sprintf("BigInteger constructor: wrong number of args (%d)", len(args)))
	}
	if len(args) == 1 {
		if text, ok := args[0].(string); ok {
			value, err := lang.NewBigInt(text)
			if err != nil {
				panic(err)
			}
			return value
		}
	}
	magnitudeArg := args[len(args)-1]
	var bytes []byte
	switch value := magnitudeArg.(type) {
	case []byte:
		bytes = append(bytes, value...)
	case []int8:
		for _, item := range value {
			bytes = append(bytes, byte(item))
		}
	case interface {
		Count() int
		Nth(int) any
	}:
		for index := 0; index < value.Count(); index++ {
			bytes = append(bytes, byte(lang.AsInt64(value.Nth(index))))
		}
	default:
		panic(fmt.Sprintf("BigInteger constructor: unsupported %T", magnitudeArg))
	}
	number := new(big.Int).SetBytes(bytes)
	if len(args) == 2 {
		signum := lang.MustAsInt(args[0])
		if signum < -1 || signum > 1 {
			panic(fmt.Sprintf("BigInteger constructor: invalid signum %d", signum))
		}
		if signum == 0 && number.Sign() != 0 {
			panic("BigInteger constructor: signum-magnitude mismatch")
		}
		if signum < 0 {
			number.Neg(number)
		}
		return lang.NewBigIntFromGoBigInt(number)
	}
	if len(bytes) > 0 && bytes[0]&0x80 != 0 {
		number.Sub(number, new(big.Int).Lsh(big.NewInt(1), uint(8*len(bytes))))
	}
	return lang.NewBigIntFromGoBigInt(number)
}

func init() {
	register("Object", "java.lang", reflect.TypeOf((*any)(nil)).Elem())
	register("Byte", "java.lang", reflect.TypeOf(int8(0)))
	pkgmap.Set("Byte.MIN_VALUE", int8(-128))
	pkgmap.Set("java.lang.Byte.MIN_VALUE", int8(-128))
	pkgmap.Set("Byte.MAX_VALUE", int8(127))
	pkgmap.Set("java.lang.Byte.MAX_VALUE", int8(127))
	pkgmap.Set("Byte.TYPE", reflect.TypeOf(int8(0)))
	pkgmap.Set("java.lang.Byte.TYPE", reflect.TypeOf(int8(0)))
	register("Short", "java.lang", reflect.TypeOf(int16(0)))
	register("Float", "java.lang", reflect.TypeOf(float32(0)))
	register("Comparable", "java.lang", reflect.TypeOf((*lang.Comparer)(nil)).Elem())
	register("CharSequence", "java.lang", reflect.TypeOf(""))
	pkgmap.SetHostClassPackage("Number", "java.lang")
	pkgmap.SetHostClass("Number", lang.NewClassWithTypes(
		"java.lang.Number", reflect.TypeOf(int(0)), reflect.TypeOf(int32(0)),
		reflect.TypeOf(int64(0)), reflect.TypeOf(float32(0)), reflect.TypeOf(float64(0)),
		reflect.TypeOf((*lang.BigInt)(nil)), reflect.TypeOf((*lang.Ratio)(nil)),
		reflect.TypeOf((*lang.BigDecimal)(nil))))
	register("Class", "java.lang", reflect.TypeOf((*reflect.Type)(nil)).Elem())
	register("ClassNotFoundException", "java.lang", reflect.TypeOf((*error)(nil)).Elem())
	register("NumberFormatException", "java.lang", reflect.TypeOf((*error)(nil)).Elem())
	register("Set", "java.util", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem())
	register("Collection", "java.util", reflect.TypeOf((*lang.IPersistentCollection)(nil)).Elem())
	register("List", "java.util", reflect.TypeOf((*lang.Sequential)(nil)).Elem())
	register("Map", "java.util", reflect.TypeOf((*lang.IPersistentMap)(nil)).Elem())
	register("Keyword", "clojure.lang", reflect.TypeOf(lang.Keyword{}))
	register("Namespace", "clojure.lang", reflect.TypeOf((*lang.Namespace)(nil)))
	register("Var", "clojure.lang", reflect.TypeOf((*lang.Var)(nil)))
	pkgmap.SetHostClassPackage("BigInteger", "java.math")
	pkgmap.SetHostClass("BigInteger", lang.NewClassWithTypes(
		"java.math.BigInteger",
		reflect.TypeOf((*big.Int)(nil)),
		reflect.TypeOf((*lang.BigInt)(nil))))
	register("BigInt", "clojure.lang", reflect.TypeOf((*lang.BigInt)(nil)))
	register("AtomicInteger", "java.util.concurrent.atomic", reflect.TypeOf(int32(0)))
	register("AtomicLong", "java.util.concurrent.atomic", reflect.TypeOf(int64(0)))
	register("PersistentVector", "clojure.lang", reflect.TypeOf((*lang.Vector)(nil)))
	bigIntegerConstructor := lang.FnFunc(newBigInteger)
	lang.RegisterHostConstructor("java.math.BigInteger", bigIntegerConstructor)
	lang.RegisterHostTypeConstructor(reflect.TypeOf((*lang.BigInt)(nil)), bigIntegerConstructor)
	lang.RegisterHostTypeConstructor(reflect.TypeOf((*big.Int)(nil)), bigIntegerConstructor)
	register("IDeref", "clojure.lang", reflect.TypeOf((*lang.IDeref)(nil)).Elem())
	register("Associative", "clojure.lang", reflect.TypeOf((*lang.Associative)(nil)).Elem())
	register("Counted", "clojure.lang", reflect.TypeOf((*lang.Counted)(nil)).Elem())
	register("IAtom", "clojure.lang", reflect.TypeOf((*lang.IAtom)(nil)).Elem())
	register("IBlockingDeref", "clojure.lang", reflect.TypeOf((*lang.IBlockingDeref)(nil)).Elem())
	register("IChunkedSeq", "clojure.lang", reflect.TypeOf((*lang.IChunkedSeq)(nil)).Elem())
	register("IEditableCollection", "clojure.lang", reflect.TypeOf((*lang.IEditableCollection)(nil)).Elem())
	register("IFn", "clojure.lang", reflect.TypeOf((*lang.IFn)(nil)).Elem())
	register("ILookup", "clojure.lang", reflect.TypeOf((*lang.ILookup)(nil)).Elem())
	register("IMapEntry", "clojure.lang", reflect.TypeOf((*lang.IMapEntry)(nil)).Elem())
	register("IMeta", "clojure.lang", reflect.TypeOf((*lang.IMeta)(nil)).Elem())
	register("IObj", "clojure.lang", reflect.TypeOf((*lang.IObj)(nil)).Elem())
	register("IPending", "clojure.lang", reflect.TypeOf((*lang.IPending)(nil)).Elem())
	register("IPersistentCollection", "clojure.lang", reflect.TypeOf((*lang.IPersistentCollection)(nil)).Elem())
	register("IPersistentList", "clojure.lang", reflect.TypeOf((*lang.IPersistentList)(nil)).Elem())
	register("IPersistentMap", "clojure.lang", reflect.TypeOf((*lang.IPersistentMap)(nil)).Elem())
	register("IPersistentSet", "clojure.lang", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem())
	register("APersistentSet", "clojure.lang", reflect.TypeOf((*lang.IPersistentSet)(nil)).Elem())
	register("IPersistentVector", "clojure.lang", reflect.TypeOf((*lang.IPersistentVector)(nil)).Elem())
	register("IRecord", "clojure.lang", reflect.TypeOf((*lang.IRecord)(nil)).Elem())
	register("IReduceInit", "clojure.lang", reflect.TypeOf((*lang.IReduceInit)(nil)).Elem())
	register("IReduce", "clojure.lang", reflect.TypeOf((*lang.IReduce)(nil)).Elem())
	register("IRef", "clojure.lang", reflect.TypeOf((*lang.IRef)(nil)).Elem())
	register("ISeq", "clojure.lang", reflect.TypeOf((*lang.ISeq)(nil)).Elem())
	register("Indexed", "clojure.lang", reflect.TypeOf((*lang.Indexed)(nil)).Elem())
	register("Named", "clojure.lang", reflect.TypeOf((*lang.Named)(nil)).Elem())
	register("Reversible", "clojure.lang", reflect.TypeOf((*lang.Reversible)(nil)).Elem())
	register("Seqable", "clojure.lang", reflect.TypeOf((*lang.Seqable)(nil)).Elem())
	register("Sequential", "clojure.lang", reflect.TypeOf((*lang.Sequential)(nil)).Elem())
	register("Sorted", "clojure.lang", reflect.TypeOf((*lang.Sorted)(nil)).Elem())
	register("Ratio", "clojure.lang", reflect.TypeOf((*lang.Ratio)(nil)))
	register("Symbol", "clojure.lang", reflect.TypeOf((*lang.Symbol)(nil)))
	register("ExceptionInfo", "clojure.lang", reflect.TypeOf((*lang.ExceptionInfo)(nil)))
	register("Atom", "clojure.lang", reflect.TypeOf((*lang.Atom)(nil)))
	register("Volatile", "clojure.lang", reflect.TypeOf((*lang.Volatile)(nil)))
}
