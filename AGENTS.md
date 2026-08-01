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
- State an optimization's eligibility rule using semantic facts, without referring to a particular workload or reproducing a large source-level expression shape. If that cannot be done clearly, treat the optimization as over-specialized.
- Recognizing a core operation is acceptable when it models that operation's general semantics. Do not make an optimization depend on a fixed chain of core operations, a short allowlist chosen from one workload, fixed collection nesting, fixed path lengths, or assumptions about otherwise-unproven parameter types.
- Prefer shared abstract domains, function summaries, effect models, and fixed-point analysis over pass-local mini-interpreters. Extend those shared mechanisms when a new optimization needs another type or ownership fact.
- Keep benchmark workloads as held-out validation. Unit tests for an optimization must include multiple structurally different positive cases, meaningful variations in arity and representation, and near-miss cases that exercise the conservative fallback; do not copy the motivating benchmark into the test.
- Require per-pass ablation evidence. Record which distinct workloads or representative microbenchmarks activate the pass, and remove or simplify a pass whose specialized portion has no material measured benefit.
- When an optimization currently supports only a bounded case, derive that bound from a representation invariant or explicit proof. Do not silently assign types to unproven values or encode arbitrary depth, arity, field, key, or path limits merely because they cover the motivating program.
- Review generated code as part of optimization review. The optimized path should look like a natural lowering of proven facts, while the guarded fallback must remain ordinary language semantics.
- Distinguish semantic facts from representation facts. Knowing that a value is an integer, vector, map, or function does not prove its concrete Go type, protocol implementation, mutability, metadata behavior, or traversal semantics; record and consume those facts separately.
- Treat every optimized precondition as a proof obligation. Document its source, the transformations that preserve it, the guard that establishes it when static proof is unavailable, and the exact optimized operation that consumes it.
- Run every speculative guard before observable work. If a guard can fail after evaluating user code, fall back using the values and effects already produced; never restart the enclosing expression or function.
- An interface assertion proves method availability, not semantic equivalence. Replacing a protocol operation with representation-specific behavior requires a closed built-in representation, an explicit runtime guard, or separately established protocol laws.
- Add adversarial proof tests for aliases, closure capture, empty and dynamic paths, metadata, custom protocol implementations, noncanonical Go numeric types, mutation during dispatch, and guards that fail after earlier effects.
