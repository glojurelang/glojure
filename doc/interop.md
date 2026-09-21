# Host interop: Go and JVM

Glojure source files use the `.glj` extension when they rely on
Glojure-only features: Go package symbols such as `fmt.Sprintf`, the `go/`
builtins and `go/go`, Go values and their methods, the `glojure.*`
namespaces, and the `#?(:glj ...)` reader feature.
A file that is also valid Clojure may use `.clj` or `.cljc` instead.
Each load-path entry is searched for `.glj`, then `.clj`, then `.cljc`.
User code is never rewritten; the rewriter only produces the bundled
Clojure standard library.
A source file can reach two hosts, Go and the JVM compatibility layer, and
the two are told apart by the shape of the symbol alone.

## The shape rule

Go host symbols name a package and an exported identifier:

- A lowercase package path, `.`, then a Capitalized identifier:
  `strings.ToUpper`, `fmt.Sprintf`, `os.Args`.
- Slashes in the package path are written as `:`: `net:http.Get`,
  `math:rand.Intn`, `github.com:foo:bar.Baz`.
- `go/` prefixes the Go builtins: `go/append`, `go/make`.

JVM host symbols name a class and a member:

- A Capitalized class, `/`, then the member: `Math/abs`, `Integer/MAX_VALUE`,
  `Boolean/parseBoolean`, `System/getenv`.
- A fully qualified dotted class path works too:
  `java.lang.Math/abs`, `java.util.regex.Pattern/compile`.
- A bare class name resolves to the class value: `Integer`, `Number`,
  `java.util.regex.Pattern`.
- Constructors use the trailing dot: `(Long. "42")`, `(Pattern. "a+")`.
- Instance members use the leading dot: `(.toUpperCase s)`,
  `(.getBytes s)`.

A dotted symbol without `:` whose first segment is `java`, `javax`,
`clojure`, `sun`, or `jdk` is always a JVM class path.
It is never treated as a Go import, in the evaluator or in AOT-compiled
code.
An unknown JVM class compiles to a runtime error naming the class instead
of failing the Go build.

## How JVM symbols resolve

The JVM layer lives in `pkg/javacompat/*`.
Each bridge registers its statics and constants in the package map under
the short class name (`Boolean.parseBoolean`), records the class's Java
package (`java.lang`), and registers a `lang.Class` value for the class.
Constructors are registered with `lang.RegisterHostConstructor`.

Both execution paths resolve `Class/member` through one function,
`pkgmap.LookupHostMember`.
It tries the exact key first and then, for a fully qualified class path
whose package matches the registered one, the short key.
The evaluator calls it directly.
The AOT code generator emits a call to it, so the two paths cannot drift.

`clojure.lang.*` statics used by core, such as
`clojure.lang.PersistentTreeMap/create`, are registered from
`pkg/lang/hostmembers.go` in the same way.

## What is not covered yet

- Go symbols written in JVM shape, such as `strings/ToUpper`, still resolve
  by accident.
  The analyzer does not yet enforce the shape rule.
- `String/format` with a `to-array` argument list and the
  `clojure.java.io` namespace are not implemented.
