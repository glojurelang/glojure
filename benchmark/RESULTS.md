# Optimization benchmark snapshot

Measured with Go 1.24.0 on an Apple M4 Max. Times are medians of repeated
benchmarks; allocations are reported by Go's benchmark harness.

| Workload | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| AOT float kernel | 232 ms, ~17.0M allocs | 1.265 ms, 1 alloc | ~183× |
| AOT reduce pipeline | 95 ms, ~5.87M allocs | 1.905 ms, 48 allocs | ~50× |
| Interpreter constant arithmetic | 113.27 ms | 101.99 ms | ~10% |
| Interpreter constant branch | 304.61 ms, ~20.0M allocs | 59.45 ms, 6 allocs | ~5.1× |

The AOT baselines were captured before primitive float specialization and
integer pipeline fusion. The interpreter comparison uses
`benchmark/interpreter/constant_arithmetic_test.go` immediately before and
after literal `lang.Numbers` folding.

The portable programs shared with let-go currently produce these median
wall-clock ratios, where a value below 1.0 favors Glojure:

| Program | Glojure / let-go |
| --- | ---: |
| prime workload | 0.737 |
| event analytics | 0.701 |
| game of life | 0.670 |
| Mandelbrot | 0.855 |

These numbers are a development snapshot, not portable performance
guarantees. Re-run the harnesses in `benchmark/aot`, `benchmark/interpreter`,
and `benchmark/portable` when comparing later compiler changes.
