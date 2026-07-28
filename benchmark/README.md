# Glojure benchmarks

The benchmark tree has four complementary layers. Choose the narrowest layer
that can answer the performance question:

- [`suite`](suite/README.md) contains portable Clojure programs, a generic
  cross-runtime process runner, and an in-process Glojure AOT harness. Start
  here when looking for broad compiler or runtime opportunities.
- [`interpreter`](interpreter/README.md) contains focused Go benchmarks for
  interpreter analysis and execution.
- [`aot`](aot/README.md) contains focused compiler fixtures and longer
  Glojure-versus-let-go AOT comparisons.
- [`yamlstar`](yamlstar/README.md) measures a real application and its
  dependency stack.

Portable-suite and YAMLStar wall time are the primary outcome measures for
strategic performance work. Focused benchmarks explain why those outcomes
move and protect the optimization mechanism from regressions. Allocation
counts are diagnostic evidence rather than a substitute for wall and CPU
measurements.

When changing a benchmark, preserve a deterministic correctness result and
record enough source and command information to reproduce comparisons across
revisions. Do not tune garbage-collector settings differently between
runtimes.
