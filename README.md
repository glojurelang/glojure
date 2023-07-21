# Glojure

![example workflow](https://github.com/glojurelang/glojure/actions/workflows/ci.yml/badge.svg)

<img alt="Gopher image" src="./doc/logo.png" width="512" />

*Gopher image derived from [@egonelbre](https://github.com/egonelbre/gophers), licensed under [Creative Commons 1.0 Attributions license](https://creativecommons.org/licenses/by/1.0/).*

Glojure is an interpreter for [Clojure](https://clojure.org/), hosted
in Go. Glojure provides easy access to Go libraries, just as Clojure
provides easy access to Java frameworks.

## Comparisons to other Go ports of Clojure

*If you'd like to see another port in this table, please file an issue or open a pull request!*

| Aspect      | Glojure | Joker | let-go |
| ----------- | ----------- | | |
| Header      | Title       | | |
| Paragraph   | Text        | | |

Glojure makes some fundamental design choices differently from other
ports of Clojure to Go.

First, Glojure strives to be hosted in Go in the same sense in which
Clojure is hosted on the JVM. What does it mean to be a hosted
language? For Clojure on the JVM, it means that all Java values are
also Clojure values, and vice versa. Glojure strives to maintain the
same relationship with Go.
