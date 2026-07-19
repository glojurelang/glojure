# Portable interpreter benchmarks

These workloads exercise ordinary Clojure programs that run unchanged in both
Glojure and let-go. They complement microbenchmarks by covering lazy sequences,
persistent maps and vectors, higher-order collection operations, nested
function calls, and integer-heavy loops.

- `prime-workload.clj`: lazy prime generation and repeated factorization
- `event-analytics.clj`: aggregation of 75,000 event maps
- `game-of-life.clj`: 75 generations on a 48×48 board
- `mandelbrot.clj`: fixed-point Mandelbrot calculation on a 120×72 grid

Build Glojure, then compare it with a let-go executable:

```sh
go build -o /tmp/glj ./cmd/glj
go run ./benchmark/portable \
  -glojure /tmp/glj \
  -let-go /path/to/lg \
  -runs 11
```

The runner warms up both executables, checks that they produce identical
output, alternates which executable runs first, and reports median wall-clock
times. A Glojure/let-go ratio at or below `1.0` means Glojure is at least as
fast for that workload.
