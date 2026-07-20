# Long-running AOT benchmarks

This suite compares generated Glojure and let-go applications after process
startup has become a small part of the measurement. It complements the short
compiler-focused benchmarks in the parent directory.

The fixtures are ordinary Clojure namespaces accepted unchanged by both
compilers:

- `batched-fib`: twenty independent `fib(35)` calls
- `compute-bound`: let-go's sequence-heavy
  `docs/perf/microbench/compute-bound.lg` workload
- `event-analytics`: three aggregations of 75,000 persistent event maps
- `game-of-life`: eight independent 75-generation simulations on a 48×48 board

Repetition lengthens the original workloads without changing their input sizes
or algorithmic shape. Every executable validates its result before printing it,
and the comparison runner also requires byte-for-byte identical output.

## Run

From the Glojure repository:

```sh
go run ./benchmark/aot/long \
  -let-go-root ../let-go \
  -runs 5
```

Use `-glojure-go` and `-let-go-go` when pinning separate toolchains. For
example, this repository's locally managed Go 1.24 compiler can be selected
with `-glojure-go .cache/local/go-1.24.0/bin/go`; let-go can still select the
newer toolchain required by its own module.

The runner:

1. generates fresh Glojure loaders in a temporary Go 1.24 module and links
   them with the compact `glj_aot_runtime` tag;
2. clones the supplied let-go checkout into the temporary directory;
3. regenerates let-go's native `gogen_ir` core and lowers the same application
   fixtures with `scripts/lg-compile`;
4. builds stripped selector executables containing all four workloads;
5. warms each executable, alternates run order, and reports median wall time.

The let-go host asserts that every `run` Var is a `vm.NativeFn`. Its generated
core-package import wireup is copied into the host so dynamic application calls
cannot silently fall back to a bytecode-only core. The Glojure host imports and
executes only freshly generated namespace loaders, without a source path.

Building let-go's native core is the slow part. Keep the temporary directory or
reuse previously built selector executables when iterating on the timing
harness:

```sh
go run ./benchmark/aot/long -keep -runs 1

go run ./benchmark/aot/long \
  -glojure-aot /path/to/glojure-long-aot \
  -let-go-aot /path/to/let-go-long-aot \
  -runs 11
```

When supplied executables are reused, they must implement this suite's
workload-name interface and expected outputs. A Glojure/let-go ratio at or
below `1.0` favors Glojure.
