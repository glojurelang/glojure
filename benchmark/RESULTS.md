# Optimization benchmark snapshot

Measured with Go 1.24.0 on an Apple M4 Max. Times are medians of repeated
benchmarks; allocations are reported by Go's benchmark harness.

| Workload | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| AOT float kernel | 232 ms, ~17.0M allocs | 1.265 ms, 1 alloc | ~183× |
| AOT reduce pipeline | 95 ms, ~5.87M allocs | 1.905 ms, 48 allocs | ~50× |
| Exact let-go map/filter, AOT hot path | 33.42 µs, 46.0 KiB, 1,208 allocs | 7.40 µs, 1.12 KiB, 49 allocs | ~4.5× |
| Boxed common-int arithmetic | 18.1 ns, 19 B, 2 allocs | 1.98 ns, 0 B, 0 allocs | ~9.1× |
| AOT Game of Life | 982.22 ms, ~31.1M allocs | 686.85 ms, ~1.57M allocs | ~1.43× |
| Interpreter constant arithmetic | 113.27 ms | 101.99 ms | ~10% |
| Interpreter constant branch | 304.61 ms, ~20.0M allocs | 59.45 ms, 6 allocs | ~5.1× |
| Exact let-go map/filter, interpreter hot path | 40.09 µs, 49.9 KiB, 1,224 allocs | 7.64 µs, 1.12 KiB, 48 allocs | ~5.2× |

The AOT baselines were captured before primitive float specialization and
integer pipeline fusion. The interpreter comparison uses
`benchmark/interpreter/constant_arithmetic_test.go` immediately before and
after literal `lang.Numbers` folding. The exact let-go rows compare commit
`eaf816e` with the guarded square/filter/take pipeline optimization using the
fixtures committed in the corresponding benchmark directories.

The portable programs shared with let-go currently produce these median
wall-clock ratios, where a value below 1.0 favors Glojure:

| Program | Glojure / let-go |
| --- | ---: |
| prime workload | 0.737 |
| event analytics | 0.701 |
| game of life | 0.670 |
| Mandelbrot | 0.855 |

The longer native-application suite produces these startup-inclusive medians:

| Workload | Glojure AOT | let-go AOT | Glojure / let-go |
| --- | ---: | ---: | ---: |
| batched `fib(35)` | 769.64 ms | 1.322 s | 0.582 |
| let-go compute-bound | 224.68 ms | 1.202 s | 0.187 |
| event analytics | 1.315 s | 2.280 s | 0.577 |
| Game of Life | 686.85 ms | 1.015 s | 0.676 |

The stripped selectors containing all four workloads are 11.94 MiB for
Glojure and 15.73 MiB for let-go. These measurements used Glojure commit
`a26005b` plus the shared boxed-integer cache change, let-go commit `9dd835e`,
Go 1.24.0 for Glojure, and let-go's required Go 1.26.0 toolchain on the same
Apple M4 Max.

These numbers are a development snapshot, not portable performance
guarantees. Re-run the harnesses in `benchmark/aot`, `benchmark/interpreter`,
and `benchmark/portable` when comparing later compiler changes.
