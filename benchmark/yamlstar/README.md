# YAMLStar AOT benchmarks

This suite compiles and benchmarks YAMLStar's real YAML 1.2 parser and load
pipeline with the Glojure checkout under test. It reports time and allocations
for the four inputs from YAMLStar's own benchmark:

- `scalar`: a plain scalar
- `mapping`: `foo: 42`
- `nested`: nested mappings and a block sequence
- `types`: strings, integers, floats, booleans, and nulls

Each input has two benchmarks:

- `Parse` measures YAML text to parser events.
- `Load` measures the complete YAMLStar path from text to Clojure data,
  including parsing, composition, resolution, and construction.

The runner uses the last YAMLStar revision before its parser moved to a Maven
artifact (`64d1d0dd786877d2f0d8a96ccf7cd7905a8dc895`). This keeps the workload
source-based and reproducible for Glojure AOT compilation. It applies two
documented compatibility rewrites to the temporary source copy: JVM
`Exception` becomes Glojure's `go/any`, and an ASCII-irrelevant
`unicode/utf8` interop call uses Glojure's character representation instead.

## Run

From the Glojure repository:

```sh
go run ./benchmark/yamlstar -count 5 -benchtime 2s
```

The default run clones the pinned public YAMLStar revision into its temporary
build directory. To avoid network access, prepare that revision locally and
pass it to the runner:

```sh
git clone https://github.com/yaml/yamlstar.git /tmp/yamlstar
git -C /tmp/yamlstar checkout 64d1d0dd786877d2f0d8a96ccf7cd7905a8dc895

go run ./benchmark/yamlstar \
  -yamlstar-root /tmp/yamlstar \
  -count 5 \
  -benchtime 2s
```

Use `-bench` to select a subset, for example:

```sh
go run ./benchmark/yamlstar -yamlstar-root /tmp/yamlstar -bench 'Load/(nested|types)$'
```

The runner first checks the four loaded values for correctness. Namespace
loading and the first invocation happen before benchmark timing begins, so the
numbers focus on the generated application path rather than compiler or
process startup.
