# Glojure

Glojure is an interpreter for [Clojure](https://clojure.org/), hosted
in Go. Glojure provides easy access to Go libraries, just as Clojure
provides easy access to Java frameworks.

## Frequently Asked Questions

(TODO)

**How does Glojure compare to Joker?**

Glojure makes some fundamental design choices differently from Joker.

First, Glojure strives to be hosted in Go in the same sense in which
Clojure is hosted on the JVM. What does it mean to be a hosted
language? For Clojure on the JVM, it means that all Java values are
also Clojure values, and vice versa. Glojure strives to maintain the
same relationship with Go.


Less importantly, the Glojure project was begun before its primary
author was aware of other projects bringing Clojure to Go.

**Why not a compiler?**


## Golang Interop

### Accessing Exported Go Values

Symbols in the `go/` namespace are resolved to the values of the
corresponding exported Go symbols. The fully-qualified name of the
symbol (that is, including the package name), should follow `go/`. For
example:

```
-> strings.HasPrefix
func(string, string) bool
```

### Field and Method Access


### Calling Glojure from Go
