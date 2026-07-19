# Interpreter optimization benchmarks

These Go benchmarks load Glojure source once, outside the timed region, then
measure repeated execution through the interpreter:

```sh
go test ./benchmark/interpreter -bench . -benchmem -count 5 -benchtime 2s
```

`constant-arithmetic` measures a numeric loop containing a constant
subexpression through a function call. It is intended to show whether shared
compiler optimizations improve interpreted execution rather than parsing or
startup.
