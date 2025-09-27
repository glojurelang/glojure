# Glojure Test Runner

A test runner for Glojure, inspired by cognitect-labs/test-runner.

## Features

- **Test Discovery**: Automatically finds and runs test namespaces
- **Namespace Filtering**: Filter tests by regex pattern
- **Multiple Output Formats**: Console (with colors), TAP, JSON, EDN
- **Flexible Configuration**: Command-line flags for various options
- **Makefile Integration**: Easy to use with make commands

## Usage

### Basic Commands

```bash
# Run all tests
go run ./cmd/glj-test --dir test/glojure

# Run tests matching a pattern
go run ./cmd/glj-test --dir test/glojure --namespace ".*basic.*"

# List test namespaces without running
go run ./cmd/glj-test --dir test/glojure --list

# Verbose output
go run ./cmd/glj-test --dir test/glojure --verbose
```

### Output Formats

```bash
# Console format (default, with colors)
go run ./cmd/glj-test --dir test/glojure --format console

# TAP format
go run ./cmd/glj-test --dir test/glojure --format tap

# JSON format
go run ./cmd/glj-test --dir test/glojure --format json

# EDN format
go run ./cmd/glj-test --dir test/glojure --format edn
```

### Makefile Targets

```bash
# Run all tests
make test-runner

# Run specific namespace tests
make test-ns NS=basic
make test-ns NS=string
```

## Command-Line Options

- `--dir`: Test directories (comma-separated, default: "test")
- `--namespace`: Namespace pattern (regex)
- `--format`: Output format (console, tap, json, edn)
- `--output`: Output file (stdout if not specified)
- `--fail-fast`: Stop on first failure
- `--include`: Include tests with metadata (comma-separated)
- `--exclude`: Exclude tests with metadata (comma-separated)
- `--parallel`: Number of parallel test runners (default: 1)
- `--verbose`: Verbose output
- `--list`: List test namespaces without running

## Test Discovery

The test runner automatically discovers test namespaces by:
1. Finding all `.glj` files in specified directories
2. Reading the namespace declaration from each file
3. Filtering for namespaces containing "test" in their name

## Output Examples

### Console Format
```
Ran 7 tests containing 31 assertions.
✓ 31 passed

SUCCESS!
```

### JSON Format
```json
{"tests":7,"passed":31,"failed":0,"errors":0,"success":true}
```

### TAP Format
```
TAP version 13
1..31
# tests 7
# pass 31
```