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

- `constant-arithmetic.glj`: repeated arithmetic containing a constant
  subexpression, through a direct function call.
- `float-kernel.glj`: repeated floating-point arithmetic through a direct
  function call.
- `letgo-map-filter.glj`: let-go's exact map/filter/take benchmark, including
  its anonymous integer-square callback.
- `reduce-pipeline.glj`: reduction over lazy `map` and `filter` producers.

Run a baseline before changing the compiler and compare it with the same
command afterward. The reported allocation counts are particularly useful for
checking unboxing and sequence fusion.

For startup-inclusive, cross-runtime workloads designed to run for roughly one
second or longer in at least one runtime, use the
[long-running AOT suite](long/README.md). It builds the same Clojure fixtures
as native Glojure and let-go applications and compares their median wall times.
