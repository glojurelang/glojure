# AOT compiler benchmarks

These benchmarks measure generated Glojure code without including process
startup, source parsing, or namespace loading in the timed region. They are
intended to expose compiler optimizations that the portable end-to-end
benchmarks cannot isolate.

The runner creates a temporary Go module, copies the fixtures, generates fresh
AOT loaders with the current compiler, and runs Go benchmarks against the
loaded `run` Vars:

```sh
go run ./benchmark/aot -count 5 -benchtime 2s
```

Use `-glojure /path/to/glj` to reuse an existing bootstrap compiler executable.
By default, the runner builds one from the current checkout with
`glj_no_aot_stdlib`. Use `-keep` to retain the temporary module for inspection.

The fixtures deliberately cover optimization gaps:

- `float-kernel.glj`: repeated floating-point arithmetic through a direct
  function call.
- `reduce-pipeline.glj`: reduction over lazy `map` and `filter` producers.

Run a baseline before changing the compiler and compare it with the same
command afterward. The reported allocation counts are particularly useful for
checking unboxing and sequence fusion.
