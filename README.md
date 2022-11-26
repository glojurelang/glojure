# Glojure

Glojure is an interpreter for [Clojure](https://clojure.org/), hosted
in Go. Glojure provides easy access to Go libraries, just as Clojure
provides easy access to Java frameworks.

## Frequently Asked Questions

(TODO)

- *Why not a compiler?*
  - ...


## Golang Interop

### Accessing Exported Go Values

Symbols in the `go/` namespace are resolved to the values of the
corresponding exported Go symbols. The fully-qualified name of the
symbol (that is, including the package name), should follow `go/`. For
example:

```
-> go/strings.HasPrefix
<func(string, string) bool Value>
```

### Field and Method Access


### Calling Glojure from Go
