# Glojure portable benchmark suite

This directory contains implementation-neutral Clojure workloads for guiding
Glojure performance work. The suite measures language implementations, not
benchmark harness overhead inside a particular host.

Each workload:

- is ordinary Clojure source without Glojure reader conditionals;
- exposes a zero-argument `run` function and has no top-level benchmark work;
- returns deterministic data recorded in `manifest.json`;
- exercises a documented group of language and runtime features.

The initial workloads are original, portable implementations of established
benchmark families. They take inspiration from the Computer Language
Benchmarks Game and the older Clojure benchmark suites, but do not copy their
host-specific harnesses or source.

## Corpus

| Workload | Primary coverage |
| --- | --- |
| `binary-trees` | recursive allocation, records, keyword field lookup |
| `event-analytics` | nested persistent maps, keywords, `update-in` |
| `fannkuch-redux` | vector updates, indexed access, nested integer loops |
| `fasta` | deterministic generation, strings, characters, frequencies |
| `game-of-life` | vector traversal, nested calls, simulation |
| `k-nucleotide` | substrings, persistent maps, reduction, sorting |
| `mandelbrot` | nested numeric loops and function calls |
| `multimethod-dispatch` | multimethod dispatch over composite values |
| `n-body` | floating-point math and persistent vector updates |
| `persistent-map` | HAMT construction, lookup, update, and removal |
| `prime-workload` | lazy sequences, factorization, `mapcat`, reduction |
| `protocol-dispatch` | protocol extension and dynamic dispatch |
| `regex-dna` | regular expressions and string replacement |
| `sorting` | large vector construction and comparator-driven sorting |
| `spectral-norm` | floating-point kernels and nested function calls |
| `transducers` | composed transducers and reducing functions |

## End-to-end comparison

Build Glojure, then name any script-compatible Clojure executables to compare:

```sh
go build -o /tmp/glj ./cmd/glj
go run ./benchmark/suite \
  -runtime glojure=/tmp/glj \
  -runtime clojure=clojure \
  -runs 9
```

The value after `=` is a CSV command. This permits fixed arguments while
preserving spaces in paths:

```sh
go run ./benchmark/suite \
  -runtime 'clojure=clojure,-J-Xmx2g' \
  -runtime 'candidate=/tmp/Glojure candidate/glj'
```

For a runtime needing a different command-line convention, provide a small
wrapper executable which accepts one Clojure script path.

Before timing, the runner checks every runtime's printed result against the
manifest. It warms each runtime, rotates execution order between samples, and
reports median wall, user CPU, system CPU, and p95 wall time. The ratio is
relative to the first `-runtime`; values below `1.0` are faster than that
baseline. Use `-format json` to retain all raw samples for later analysis.

Useful options:

```sh
go run ./benchmark/suite -list
go run ./benchmark/suite -workload 'binary|map' ...
go run ./benchmark/suite -runs 1 -warmup 0 ...
```

These are process-level measurements, so they include startup, source loading,
analysis, and execution. Use the AOT harness for compiler-generated code
without those costs:

```sh
go run ./benchmark/suite/aot -count 5 -benchtime 2s
```

## Design rules

Keep the suite portable enough to run unchanged on JVM Clojure and developing
Clojure implementations:

- Prefer Clojure data structures and standard functions over Java interop.
- Do not add implementation-specific type hints or reader conditionals.
- Return small checksums or summaries rather than timing printing.
- Record why a workload exists through its manifest tags.
- Add a correctness expectation before using a workload for performance work.
- Tune standard workloads to be long enough to dominate timer noise without
  making the whole suite prohibitively slow.

Microbenchmarks still belong near the implementation they diagnose. YAMLStar
remains the representative application benchmark. This suite fills the gap
between those two: recognizable, independently runnable Clojure programs with
enough breadth to reveal strategic compiler and runtime wins.
