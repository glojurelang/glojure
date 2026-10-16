# Interpreter optimization benchmarks

These Go benchmarks measure repeated execution through the interpreter. Most
load Glojure source once outside the timed region; benchmarks whose names
start with `Load` include reading, analysis, and execution:

```sh
go test ./benchmark/interpreter -bench . -benchmem -count 5 -benchtime 2s
```

`constant-arithmetic` measures a numeric loop containing a constant
subexpression through a function call. It is intended to show whether shared
compiler optimizations improve interpreted execution rather than parsing or
startup.

`constant-branch` measures whether a literal predicate inside a numeric loop
prevents the interpreter's existing typed-loop compiler from recognizing the
surrounding loop.

`let-go-map-filter` and `let-go-tak` reproduce the two official let-go
interpreter workloads that are closest to Glojure. Keeping them in-process
removes executable startup noise while profiling and validating changes.
`BenchmarkLoadLetGoMapFilter` separately exercises the cold source path used
by short-lived CLI programs.

## Weird numbers and CLI comparisons

The weird numbers fixture is the numeric Clojure equivalent of YAMLScript's
Rosetta Code sample.
It preserves the sieve, divisor search, and recursive subset search.
The execution benchmark loads its definitions before timing; the load benchmark
measures source evaluation separately.

Run with the project's Makes-provided Go:

```bash
make shell CMD='go test ./benchmark/interpreter -run TestWeirdNumbers'
make shell CMD='go test ./benchmark/interpreter -run "^$" \
  -bench WeirdNumbers -benchmem -cpuprofile .cache/weird.cpu \
  -memprofile .cache/weird.mem'
```

For complete YAMLScript CLI measurements, preserve a reference binary and use:

```bash
python3 benchmark/interpreter/compare_cli.py \
  /path/to/reference-ys /path/to/candidate-ys /path/to/graalvm-ys \
  /path/to/weird-numbers.ys --output /path/to/results.json
```

The runner performs two warmups and eleven timed invocations per binary,
rotates execution order, verifies identical output, and records elapsed times,
medians, speed ratios, binary hashes, and the input hash.
Keep toolchain versions and build options fixed and avoid concurrent builds or
tests while measuring.
Confirm gains in a second batch before accepting a performance milestone.

## Training a generated CLI

`profile_cli.py` adds a temporary Go benchmark to an existing Gloat-generated
CLI module, runs its `main` repeatedly on the supplied source files, and removes
the benchmark afterwards.
Use a development module that replaces Glojure with the checkout being tested.
The CLI must return normally from `main` for the selected inputs.

```bash
make shell CMD='python3 benchmark/interpreter/profile_cli.py \
  /path/to/generated-cli /path/to/weird-numbers.ys \
  --output /path/to/default.pgo'
```

Training disables existing PGO and uses `-trimpath`.
The adjacent JSON file records the Go version, command, input hashes, and profile
hash for the training run.
Use `go build -pgo=/path/to/default.pgo` to build a candidate with the profile.
Measure that build separately from source-only improvements, and keep the Go
version, runtime environment, and remaining flags fixed.
