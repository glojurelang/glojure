# Repository guidance

## Performance optimization

- Prefer small, composable optimizations based on general facts such as inferred type, ownership, escape behavior, call target, and concrete representation.
- Do not recognize benchmark names, function names, namespaces, or large benchmark-specific AST shapes.
- Put reusable analysis in the shared IR when practical. Interpreter and AOT backends should consume the same facts when the optimization applies to both.
- Preserve a conservative semantic fallback whenever required facts cannot be proven.
- Type-specific fast paths are acceptable when selected through general type or representation facts and useful beyond one workload.
- Before implementing an expensive compiler pass, prototype its intended output by editing generated Go and measure whether it materially improves performance.
- Validate optimizations with ablation measurements, multi-run benchmarks, the broader benchmark suite, conformance tests, and comparisons against main.
- Treat allocations and profiles as diagnostic evidence, and measure changes with reproducible benchmarks.
- Do not improve benchmark results by changing GC settings or other benchmark-only runtime configuration.
- Compare relevant behavior with Clojure's implementation, while adapting the design to Glojure and Go rather than copying it mechanically.
