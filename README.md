# Glojure

![example workflow](https://github.com/glojurelang/glojure/actions/workflows/ci.yml/badge.svg)

<img alt="Gopher image" src="./doc/logo.png" width="512" />

*Gopher image derived from [@egonelbre](https://github.com/egonelbre/gophers), licensed under [Creative Commons 1.0 Attributions license](https://creativecommons.org/licenses/by/1.0/).*

Glojure is an interpreter for [Clojure](https://clojure.org/), hosted
in Go. Glojure provides easy access to Go libraries, just as Clojure
provides easy access to Java frameworks.

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
