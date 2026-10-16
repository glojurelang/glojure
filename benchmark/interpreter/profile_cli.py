#!/usr/bin/env python3
"""Train a Go PGO profile by executing an existing generated CLI module."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("module", type=Path)
    parser.add_argument("sources", type=Path, nargs="+")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--iterations", type=int, default=12)
    parser.add_argument("--go", default="go")
    parser.add_argument("--tags", default="glj_aot_runtime,glj_no_goimports")
    args = parser.parse_args()
    if args.iterations < 1:
        parser.error("iterations must be positive")
    module = args.module.resolve()
    sources = [path.resolve(strict=True) for path in args.sources]
    output = args.output.resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    inputs = ",\n".join(json.dumps(str(path), ensure_ascii=False) for path in sources)
    fixture = module / "glojure_profile_test.go"
    # Exclusive creation protects any existing project test fixture.
    with fixture.open("x") as stream:
        stream.write('''package main

import (
    "os"
    "testing"
)

func BenchmarkGlojureCLIProfile(b *testing.B) {
    sources := []string{
''' + inputs + ''',
    }
    stdout, stderr, argv := os.Stdout, os.Stderr, os.Args
    sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
    if err != nil { b.Fatal(err) }
    os.Stdout, os.Stderr = sink, sink
    b.Cleanup(func() {
        os.Stdout, os.Stderr, os.Args = stdout, stderr, argv
        sink.Close()
    })
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for _, source := range sources {
            os.Args = []string{"ys", source}
            main()
        }
    }
}
''')
    try:
        with tempfile.TemporaryDirectory(prefix="glojure-pgo-") as temporary:
            executable = Path(temporary) / "cli.test"
            command = [
                args.go, "test", "-trimpath", "-pgo=off",
                "-tags", args.tags, "-run", "^$",
                "-bench", "^BenchmarkGlojureCLIProfile$",
                "-benchtime", f"{args.iterations}x",
                "-cpuprofile", str(output), "-o", str(executable),
            ]
            subprocess.run(command, cwd=module, check=True)
    finally:
        fixture.unlink()
    metadata = {
        "go": subprocess.check_output(
            [args.go, "version"], text=True
        ).strip(),
        "command": command,
        "environment": {key: os.environ[key] for key in
                        ("GOFLAGS", "CGO_ENABLED") if key in os.environ},
        "sources": [{"path": str(path), "sha256": hashlib.sha256(
            path.read_bytes()).hexdigest()} for path in sources],
        "profile_sha256": hashlib.sha256(output.read_bytes()).hexdigest(),
    }
    output.with_suffix(output.suffix + ".json").write_text(
        json.dumps(metadata, indent=2) + "\n"
    )


if __name__ == "__main__":
    main()
