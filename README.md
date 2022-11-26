# Glojure

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
