# Glojure

![example workflow](https://github.com/glojurelang/glojure/actions/workflows/ci.yml/badge.svg)

<img alt="Gopher image" src="./doc/logo.png" width="512" />

*Gopher image derived from [@egonelbre](https://github.com/egonelbre/gophers), licensed under [Creative Commons 1.0 Attributions license](https://creativecommons.org/licenses/by/1.0/).*

Glojure is an interpreter for [Clojure](https://clojure.org/), hosted
in Go. Glojure provides easy access to Go libraries, just as Clojure
provides easy access to Java frameworks.

**Glojure is in early development; expect bugs, missing features, and limited performance.** That
said, it is used successfully in hobby projects and runs a significant subset of the (transformed)
core Clojure library.

## Installation

Glojure is currently only available from source, and it requires at least go 1.19.
Install it with the `go install` command:
```
$ go install github.com/glojurelang/glojure/cmd/glj@latest
```

Then you can start the REPL with:
```
$ glj
user=> (println "Hello, world!")
Hello, world!
nil
user=>
```

## Interop

Glojure ships with interop with many standard library packages out-of-the-box.
Go package names are munged to avoid ambiguity with the use of `/` to refer to
namespaced symbols; instances of `/` in package names are replaced with `$`. Here's
a simple example:

```clojure
user=> (println (fmt.Sprintf "A couple of HTTP methods: %v" [net$http.MethodGet net$http.MethodPost]))
A couple of HTTP methods: ["GET" "POST"]
nil
```

The following standard library packages are included by default:
- `bytes`
- `context`
- `errors`
- `flag`
- `fmt`
- `io`
- `io/fs`
- `io/ioutil`
- `math`
- `math/big`
- `math/rand`
- `net/http`
- `os`
- `os/exec`
- `os/signal`
- `regexp`
- `reflect`
- `sort`
- `strconv`
- `strings`
- `time`
- `unicode`

To expose additional packages, you must generate a "package map" and compile your own executable
that imports both your package map and the Glojure API. See the section below for more details.

Expect improvements to both the availability of standard library packages and interop workflows.

### Accessing additional Go packages

The `gen-import-interop` can be used to emit the contents of a .go file
that will export a function that can be used to add the exports of
additional packages to the Glojure package map.

```
$ go run github.com/glojurelang/glojure/cmd/gen-import-interop \
     -packages=:comma-separated-package-list: \
     > your/package/gljimports/my_package_map.go
```

Then, in your own program:

```go
package main

import (
	"your.package/gljimports"

	"github.com/glojurelang/glojure/runtime"
)

func init() {
	gljimports.RegisterImports(pkgmap.Set)
}
```

## Comparisons to other Go ports of Clojure

*If you'd like to see another port in this table, or if you believe there is an
error in it, please file an issue or open a pull request!*

| Aspect      | Glojure | [Joker](https://github.com/candid82/joker) | [let-go](https://github.com/nooga/let-go) |
| ----------- | ----------- |----------- | -----------|
| Hosted[^1]  | Yes       | No  | No  |
| Execution   | Tree-walk interpreter | Tree-walk interpreter  | Bytecode Interpreter |
| Easy Go interop | Yes | No | No |

[^1]: What does it mean to be a hosted
language? For Clojure on the JVM, it means that all Java values are
also Clojure values, and vice versa. Glojure strives to maintain the
same relationship with Go.
