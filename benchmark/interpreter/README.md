# Interpreter optimization benchmarks

These Go benchmarks measure repeated execution through the interpreter. Most
load Glojure source once outside the timed region; benchmarks whose names
start with `Load` include reading, analysis, and execution:

```sh
go test ./benchmark/interpreter -bench . -benchmem -count 5 -benchtime 2s
```

`constant-arithmetic` measures a numeric loop containing a constant
subexpression through a function call. It is intended to show whether shared
compiler optimizations improve interpreted execution rather than parsing or
startup.

`constant-branch` measures whether a literal predicate inside a numeric loop
prevents the interpreter's existing typed-loop compiler from recognizing the
surrounding loop.

`let-go-map-filter` and `let-go-tak` reproduce the two official let-go
interpreter workloads that are closest to Glojure. Keeping them in-process
removes executable startup noise while profiling and validating changes.
`BenchmarkLoadLetGoMapFilter` separately exercises the cold source path used
by short-lived CLI programs.
